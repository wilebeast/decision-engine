package engine

import (
	"fmt"
	"maps"
	"sort"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/compiler"
	"github.com/wilebeast/decision-engine/factor"
)

func Compile(config Config) (Snapshot, error) {
	factorTypes := make(map[string]any, len(config.Factors))
	factorCodes := make(map[string]struct{}, len(config.Factors))
	for _, definition := range config.Factors {
		if definition.Code == "" {
			return Snapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "factor code is required")
		}
		if _, exists := factorCodes[definition.Code]; exists {
			return Snapshot{}, gerror.NewCodef(gcode.CodeInvalidParameter, "duplicate factor code: %s", definition.Code)
		}
		factorCodes[definition.Code] = struct{}{}
		zeroValue, err := factor.ZeroValue(definition.ValueType)
		if err != nil {
			return Snapshot{}, err
		}
		factorTypes[definition.Code] = zeroValue
	}
	closeCompiler := compiler.NewCompiler(factorTypes, config.Functions)
	defer closeCompiler()
	factorsByCode := make(map[string]factor.Factor, len(config.Factors))
	for _, definition := range config.Factors {
		runtimeFactor, err := factor.BuiltinRegistry().NewFactor(definition)
		if err != nil {
			return Snapshot{}, gerror.WrapCodef(gcode.CodeInvalidParameter, err, "create factor %s", definition.Code)
		}
		factorsByCode[definition.Code] = runtimeFactor
	}
	// Check unused definitions too; a dormant cycle must not enter a snapshot.
	if _, err := orderFactors(factorCodes, factorsByCode); err != nil {
		return Snapshot{}, err
	}

	events := make(map[string]eventSnapshot, len(config.Events))
	for _, event := range config.Events {
		if event.EventCode == "" {
			return Snapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "event code is required")
		}
		if _, exists := events[event.EventCode]; exists {
			return Snapshot{}, gerror.NewCodef(gcode.CodeInvalidParameter, "duplicate event code: %s", event.EventCode)
		}
		snapshot, err := compileEvent(event, factorsByCode)
		if err != nil {
			return Snapshot{}, gerror.WrapCodef(gcode.CodeInvalidParameter, err, "compile event %s", event.EventCode)
		}
		snapshot.functions = maps.Clone(config.Functions)
		events[event.EventCode] = snapshot
	}
	return Snapshot{events: events}, nil
}

func compileEvent(event EventDefinition, factorsByCode map[string]factor.Factor) (eventSnapshot, error) {
	if len(event.Strategies) == 0 {
		return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "event must contain strategies")
	}
	strategyIDs := make(map[string]struct{}, len(event.Strategies))
	rootFactors := make(map[string]struct{})
	strategies := make([]compiledStrategy, 0, len(event.Strategies))
	for _, strategy := range event.Strategies {
		if strategy.StrategyCode == "" || strategy.Rule.RuleCode == "" || strategy.Rule.Expression == "" || strategy.Decision.DecisionCode == "" || strategy.Action.ActionCode == "" {
			return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "strategy id, rule, decision and action are required")
		}
		if _, exists := strategyIDs[strategy.StrategyCode]; exists {
			return eventSnapshot{}, gerror.NewCodef(gcode.CodeInvalidParameter, "duplicate strategy code: %s", strategy.StrategyCode)
		}
		strategyIDs[strategy.StrategyCode] = struct{}{}
		rule, err := compileRule(strategy.Rule.Expression)
		if err != nil {
			return eventSnapshot{}, gerror.WrapCodef(gcode.CodeInvalidParameter, err, "compile rule %s", strategy.Rule.RuleCode)
		}
		// Constant rules are valid for configured collection actions.
		for _, factorCode := range rule.directFactorCodes {
			if _, ok := factorsByCode[factorCode]; !ok {
				return eventSnapshot{}, gerror.NewCodef(gcode.CodeInvalidParameter, "rule %s references unknown factor %s", strategy.Rule.RuleCode, factorCode)
			}
			rootFactors[factorCode] = struct{}{}
		}
		rule.ruleCode = strategy.Rule.RuleCode
		if !validDecision(strategy.Decision.DecisionCode) {
			return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "unknown decision")
		}
		item := compiledStrategy{strategyCode: strategy.StrategyCode, rule: rule, decisionCode: strategy.Decision.DecisionCode, priority: strategy.Decision.Priority, action: cloneAction(strategy.Action)}
		if raw, exists := strategy.Action.ActionParams["options"]; exists {
			options, ok := raw.([]any)
			if !ok || len(options) == 0 {
				return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "options must be a non-empty list")
			}
			for _, raw := range options {
				option, ok := raw.(map[string]any)
				if !ok {
					return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "invalid option")
				}
				expression := "true"
				if raw, exists := option["condition"]; exists {
					var ok bool
					expression, ok = raw.(string)
					if !ok || expression == "" {
						return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "invalid option condition")
					}
				}
				condition, err := compileRule(expression)
				if err != nil {
					return eventSnapshot{}, err
				}
				for _, code := range condition.directFactorCodes {
					rootFactors[code] = struct{}{}
				}
				item.options = append(item.options, condition)
			}
		}
		// Action payload fields may carry their own selection conditions.
		if raw, exists := strategy.Action.ActionParams["fields"]; exists {
			fields, ok := raw.([]any)
			if !ok {
				return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "fields must be a list")
			}
			seen := map[string]bool{}
			for _, raw := range fields {
				field, ok := raw.(map[string]any)
				if !ok {
					return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "invalid conditional field")
				}
				key, _ := field["field_key"].(string)
				expression, _ := field["condition"].(string)
				if key == "" || seen[key] {
					return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "invalid or duplicate field key")
				}
				seen[key] = true
				condition, err := compileRule(expression)
				if err != nil {
					return eventSnapshot{}, err
				}
				for _, code := range condition.directFactorCodes {
					rootFactors[code] = struct{}{}
				}
				item.fields = append(item.fields, compiledField{key: key, condition: condition})
			}
		}
		for _, branch := range strategy.Decision.Cases {
			if !validDecision(branch.DecisionCode) {
				return eventSnapshot{}, gerror.NewCode(gcode.CodeInvalidParameter, "unknown branch decision")
			}
			condition, err := compileRule(branch.Condition)
			if err != nil {
				return eventSnapshot{}, err
			}
			for _, code := range condition.directFactorCodes {
				rootFactors[code] = struct{}{}
			}
			item.cases = append(item.cases, compiledDecisionCase{condition: condition, decisionCode: branch.DecisionCode})
		}
		strategies = append(strategies, item)
	}

	orderedCodes, err := orderFactors(rootFactors, factorsByCode)
	if err != nil {
		return eventSnapshot{}, err
	}
	factors := make([]factor.Factor, 0, len(orderedCodes))
	for _, code := range orderedCodes {
		factors = append(factors, factorsByCode[code])
	}
	return eventSnapshot{eventCode: event.EventCode, factors: factors, strategies: strategies}, nil
}

func orderFactors(roots map[string]struct{}, factorsByCode map[string]factor.Factor) ([]string, error) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	ordered := make([]string, 0, len(roots))
	var visit func(string) error
	visit = func(code string) error {
		if visited[code] {
			return nil
		}
		if visiting[code] {
			return gerror.NewCodef(gcode.CodeInvalidParameter, "factor dependency cycle at %s", code)
		}
		runtimeFactor, ok := factorsByCode[code]
		if !ok {
			return gerror.NewCodef(gcode.CodeInvalidParameter, "unknown factor %s", code)
		}
		visiting[code] = true
		dependencies := append([]string(nil), runtimeFactor.DependentFactorCodes()...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[code] = false
		visited[code] = true
		ordered = append(ordered, code)
		return nil
	}
	codes := make([]string, 0, len(roots))
	for code := range roots {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if err := visit(code); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func compileRule(expression string) (compiledRule, error) {
	compiledExpression, err := compiler.CompileRule(expression)
	if err != nil {
		return compiledRule{}, err
	}
	return compiledRule{directFactorCodes: compiledExpression.DirectDependentFactorCodes, program: compiledExpression.Program}, nil
}

func factorError(code, errorCode string, err error) FactorResult {
	return FactorResult{Code: code, Status: FactorStatusError, ErrorCode: errorCode, ErrorMessage: fmt.Sprint(err)}
}

package engine

import (
	"context"
	"maps"
	"time"

	"github.com/expr-lang/expr"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/model"
)

func validDecision(code string) bool { return code == "pass" || code == "continue" || code == "block" }

func cloneAction(action risk.Action) risk.Action {
	action.ActionParams = cloneParams(action.ActionParams)
	return action
}
func cloneParams(params map[string]any) map[string]any {
	result := maps.Clone(params)
	for key, value := range result {
		switch v := value.(type) {
		case map[string]any:
			result[key] = cloneParams(v)
		case []string:
			result[key] = append([]string(nil), v...)
		case []any:
			items := make([]any, len(v))
			for i, item := range v {
				items[i] = cloneParams(map[string]any{"v": item})["v"]
			}
			result[key] = items
		}
	}
	return result
}

// Execute evaluates one complete event (product), not a single strategy.
// No action is executed here. Errors never become a business pass.
func (s Snapshot) Execute(ctx context.Context, input RiskInput) (DecisionResult, error) {
	if input.UniqueID == "" || input.EventCode == "" || input.ObjectType == "" {
		return DecisionResult{}, gerror.NewCode(gcode.CodeInvalidParameter, "evaluation id, event and subject type are required")
	}
	event, ok := s.events[input.EventCode]
	if !ok {
		return DecisionResult{}, gerror.NewCode(gcode.CodeNotFound, "event is not configured")
	}
	state := newRuntimeState(input.Param, event.factors, event.strategies)
	state.paramReader = input.ParamReader
	result := DecisionResult{UniqueID: input.UniqueID, EventCode: input.EventCode, Subject: risk.Subject{ObjectType: input.ObjectType, ObjectID: input.ObjectID}, Actions: []risk.Action{}}
	finish := func() DecisionResult {
		result.FactorResults = state.FactorSnapshot()
		result.StrategyResults = state.StrategySnapshot()
		return result
	}
	for _, f := range event.factors {
		if err := ctx.Err(); err != nil {
			return finish(), err
		}
		value := f.Fetch(ctx, state)
		if err := state.CompleteFactor(f.Code(), value); err != nil {
			return finish(), err
		}
		if value.Status == FactorStatusError {
			return finish(), gerror.NewCode(gcode.CodeInternalError, "required factor could not be evaluated")
		}
	}
	state.MaterializeFactorValues()
	values := state.FactorValues()
	for key, fn := range event.functions {
		values[key] = fn
	}
	for index, strategy := range event.strategies {
		if err := ctx.Err(); err != nil {
			return finish(), err
		}
		start := time.Now()
		item, _ := state.StrategyResult(index)
		item.DecisionCode = "pass"
		value, err := expr.Run(strategy.rule.program, values)
		matched, valid := value.(bool)
		if err != nil || !valid {
			item.Status = StrategyStatusExecuteFailed
			item.ErrorMessage = "rule execution failed"
			_ = state.CompleteStrategy(index, item)
			return finish(), gerror.NewCode(gcode.CodeInternalError, item.ErrorMessage)
		}
		item.IsHit = matched
		item.Status = StrategyStatusConditionNotMatched
		if matched {
			item.Status = StrategyStatusHit
			item.DecisionCode = strategy.decisionCode
			for _, branch := range strategy.cases {
				v, err := expr.Run(branch.condition.program, values)
				hit, valid := v.(bool)
				if err != nil || !valid {
					item.Status = StrategyStatusExecuteFailed
					item.ErrorMessage = "decision condition execution failed"
					_ = state.CompleteStrategy(index, item)
					return finish(), gerror.NewCode(gcode.CodeInternalError, item.ErrorMessage)
				}
				if hit {
					item.DecisionCode = branch.decisionCode
					break
				}
			}
			if item.DecisionCode == "continue" {
				action := cloneAction(strategy.action)
				if len(strategy.options) > 0 {
					selected := []any{}
					options := action.ActionParams["options"].([]any)
					for i, condition := range strategy.options {
						value, err := expr.Run(condition.program, values)
						hit, ok := value.(bool)
						if err != nil || !ok {
							item.Status = StrategyStatusExecuteFailed
							item.ErrorMessage = "option condition execution failed"
							_ = state.CompleteStrategy(index, item)
							return finish(), gerror.NewCode(gcode.CodeInternalError, item.ErrorMessage)
						}
						if hit {
							option := options[i].(map[string]any)
							delete(option, "condition")
							selected = append(selected, option)
						}
					}
					action.ActionParams["options"] = selected
				}
				if len(strategy.fields) > 0 {
					selected := []string{}
					for _, field := range strategy.fields {
						value, err := expr.Run(field.condition.program, values)
						hit, ok := value.(bool)
						if err != nil || !ok {
							item.Status = StrategyStatusExecuteFailed
							item.ErrorMessage = "action condition execution failed"
							_ = state.CompleteStrategy(index, item)
							return finish(), gerror.NewCode(gcode.CodeInternalError, item.ErrorMessage)
						}
						if hit {
							selected = append(selected, field.key)
						}
					}
					delete(action.ActionParams, "fields")
					action.ActionParams["field_keys"] = selected
				}
				result.Actions = append(result.Actions, action)
			}
		}
		item.Duration = time.Since(start)
		if err := state.CompleteStrategy(index, item); err != nil {
			return finish(), err
		}
	}
	result.DecisionCode = "pass"
	for _, item := range state.StrategySnapshot() {
		if item.DecisionCode == "block" {
			result.DecisionCode = "block"
			break
		}
		if item.DecisionCode == "continue" {
			result.DecisionCode = "continue"
		}
	}
	if result.DecisionCode == "block" {
		result.Actions = []risk.Action{}
	}
	return finish(), nil
}

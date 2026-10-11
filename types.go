package engine

import (
	"github.com/expr-lang/expr/vm"
	"github.com/wilebeast/decision-engine/factor"
	"github.com/wilebeast/decision-engine/model"
)

type ValueType = risk.ValueType
type FactorType = risk.FactorType
type FactorDefinition = risk.FactorDefinition

const (
	ValueTypeInt         = risk.ValueTypeInt
	ValueTypeFloat       = risk.ValueTypeFloat
	FactorTypeValue      = risk.FactorTypeValue
	FactorTypeExpression = risk.FactorTypeExpression
)

type RuleDefinition struct {
	RuleCode   string
	Expression string
}

type StrategyDefinition struct {
	StrategyCode string
	Rule         RuleDefinition
	Decision     DecisionDefinition
	Action       risk.Action
}

type DecisionDefinition struct {
	DecisionCode string
	Priority     int
	Cases        []DecisionCase
}

// First matching branch wins; DecisionCode remains the fallback.
type DecisionCase struct {
	Condition    string
	DecisionCode string
}

type EventDefinition struct {
	EventCode  string
	Strategies []StrategyDefinition
}

type Config struct {
	Factors []FactorDefinition
	Events  []EventDefinition
	// Trusted code only; business helpers are not external configuration.
	Functions map[string]any
}

type RiskInput = risk.RiskInput
type FactorStatus = risk.FactorStatus
type FactorResult = risk.FactorResult
type StrategyStatus = risk.StrategyStatus
type StrategyResult = risk.StrategyResult
type DecisionResult = risk.DecisionResult

const (
	FactorStatusSuccess               = risk.FactorStatusSuccess
	FactorStatusError                 = risk.FactorStatusError
	StrategyStatusPending             = risk.StrategyStatusPending
	StrategyStatusConditionNotMatched = risk.StrategyStatusConditionNotMatched
	StrategyStatusHit                 = risk.StrategyStatusHit
	StrategyStatusInputInvalid        = risk.StrategyStatusInputInvalid
	StrategyStatusExecuteFailed       = risk.StrategyStatusExecuteFailed
)

type compiledRule struct {
	ruleCode          string
	directFactorCodes []string
	program           *vm.Program
}

type compiledStrategy struct {
	strategyCode string
	rule         compiledRule
	decisionCode string
	priority     int
	action       risk.Action
	cases        []compiledDecisionCase
	fields       []compiledField
	options      []compiledRule
}
type compiledField struct {
	key       string
	condition compiledRule
}

type compiledDecisionCase struct {
	condition    compiledRule
	decisionCode string
}

type eventSnapshot struct {
	eventCode  string
	factors    []factor.Factor
	strategies []compiledStrategy
	functions  map[string]any
}

type Snapshot struct {
	events map[string]eventSnapshot
}

package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wilebeast/decision-engine/factor"
	"github.com/wilebeast/decision-engine/model"
)

func TestRuntimeStateHoldsEvaluationResults(t *testing.T) {
	input := map[string]any{"count": int64(12)}
	strategies := []compiledStrategy{{strategyCode: "strategy-1", rule: compiledRule{ruleCode: "rule-1"}}}
	state := newRuntimeState(input, []factor.Factor{runtimeTestFactor{code: "f_count"}}, strategies)
	input["count"] = int64(99)

	value, ok := state.ParamValue("count")
	assert.True(t, ok)
	assert.Equal(t, int64(12), value)
	strategyResult, err := state.StrategyResult(0)
	assert.NoError(t, err)
	assert.Equal(t, StrategyResult{StrategyCode: "strategy-1", RuleCode: "rule-1", Status: StrategyStatusPending}, strategyResult)
	assert.Equal(t, []StrategyResult{{StrategyCode: "strategy-1", RuleCode: "rule-1", Status: StrategyStatusPending}}, state.StrategySnapshot())

	assert.NoError(t, state.CompleteFactor("f_count", FactorResult{Code: "f_count", Status: FactorStatusSuccess, Value: int64(12)}))
	state.MaterializeFactorValues()
	assert.Equal(t, map[string]any{"f_count": int64(12)}, state.FactorValues())
	assert.NoError(t, state.CompleteStrategy(0, StrategyResult{StrategyCode: "strategy-1", RuleCode: "rule-1", Status: StrategyStatusHit, IsHit: true}))
	assert.Equal(t, []StrategyResult{{StrategyCode: "strategy-1", RuleCode: "rule-1", Status: StrategyStatusHit, IsHit: true}}, state.StrategySnapshot())
}

type runtimeTestFactor struct{ code string }

func (f runtimeTestFactor) Code() string { return f.code }

func (runtimeTestFactor) DependentFactorCodes() []string { return nil }

func (runtimeTestFactor) Fetch(context.Context, factor.Runtime) risk.FactorResult {
	return risk.FactorResult{}
}

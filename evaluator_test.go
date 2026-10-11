package engine

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wilebeast/decision-engine/model"
	"sync"
	"testing"
)

func testConfig() Config {
	return Config{Factors: []FactorDefinition{
		{Code: "f_count", Type: FactorTypeValue, ValueType: ValueTypeInt, ParamKey: "count"},
		{Code: "f_ratio", Type: FactorTypeExpression, ValueType: ValueTypeFloat, Expression: "safe_divide(f_count,f_count)"},
	}, Events: []EventDefinition{{EventCode: "PRODUCT", Strategies: []StrategyDefinition{
		{StrategyCode: "first", Rule: RuleDefinition{RuleCode: "rule1", Expression: "f_ratio > 0"}, Decision: DecisionDefinition{DecisionCode: "continue", Cases: []DecisionCase{{Condition: "f_count > 10", DecisionCode: "block"}}}, Action: risk.Action{ActionCode: "A", ActionParams: map[string]any{"items": []any{"one"}}}},
		{StrategyCode: "second", Rule: RuleDefinition{RuleCode: "rule2", Expression: "f_count < 100"}, Decision: DecisionDefinition{DecisionCode: "continue", Priority: 999}, Action: risk.Action{ActionCode: "B"}},
	}}}}
}
func TestAggregateAllStrategiesRatherThanPriorityWinner(t *testing.T) {
	snapshot, err := Compile(testConfig())
	require.NoError(t, err)
	input := RiskInput{UniqueID: "evaluation", EventCode: "PRODUCT", ObjectType: "CUSTOMER", ObjectID: "C", Param: map[string]any{"count": int64(1)}}
	result, err := snapshot.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "continue", result.DecisionCode)
	require.Len(t, result.Actions, 2)
	assert.Equal(t, "A", result.Actions[0].ActionCode)
	assert.Equal(t, "B", result.Actions[1].ActionCode)
	assert.Empty(t, result.StrategyCode)
	assert.Nil(t, result.Action)
	result.Actions[0].ActionParams["items"].([]any)[0] = "mutated"
	next, err := snapshot.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "one", next.Actions[0].ActionParams["items"].([]any)[0])
	input.Param["count"] = int64(11)
	result, err = snapshot.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "block", result.DecisionCode)
	assert.Empty(t, result.Actions)
	assert.Equal(t, "continue", result.StrategyResults[1].DecisionCode)
	input.Param["count"] = int64(0)
	result, err = snapshot.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Empty(t, result.DecisionCode)
	assert.Equal(t, FactorStatusError, result.FactorResults["f_ratio"].Status)
}
func TestUnmatchedPassAndUnknownEvent(t *testing.T) {
	config := testConfig()
	for i := range config.Events[0].Strategies {
		config.Events[0].Strategies[i].Rule.Expression = "false"
		config.Events[0].Strategies[i].Decision.Cases = nil
	}
	snapshot, err := Compile(config)
	require.NoError(t, err)
	result, err := snapshot.Execute(context.Background(), RiskInput{UniqueID: "E", EventCode: "PRODUCT", ObjectType: "CUSTOMER"})
	require.NoError(t, err)
	assert.Equal(t, "pass", result.DecisionCode)
	assert.Empty(t, result.FactorResults)
	_, err = snapshot.Execute(context.Background(), RiskInput{UniqueID: "E", EventCode: "UNKNOWN", ObjectType: "CUSTOMER"})
	assert.Error(t, err)
}
func TestConcurrentCompileSessions(t *testing.T) {
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() { defer group.Done(); _, err := Compile(testConfig()); assert.NoError(t, err) }()
	}
	group.Wait()
}

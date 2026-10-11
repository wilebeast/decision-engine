package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wilebeast/decision-engine/model"
)

func optionConfig(condition any) Config {
	return Config{Factors: []FactorDefinition{{Code: "f_count", Type: FactorTypeValue, ValueType: ValueTypeInt, ParamKey: "count"}}, Events: []EventDefinition{{EventCode: "PRODUCT", Strategies: []StrategyDefinition{{
		StrategyCode: "verification", Rule: RuleDefinition{RuleCode: "required", Expression: "true"}, Decision: DecisionDefinition{DecisionCode: "continue"},
		Action: risk.Action{ActionCode: "complete_verification", ActionParams: map[string]any{"options": []any{
			map[string]any{"condition": condition, "level_name": "conditional"}, map[string]any{"level_name": "always"},
		}}},
	}}}}}
}

func TestConditionalOptionsUseFactorDependenciesAndDoNotMutateSnapshot(t *testing.T) {
	config := optionConfig("f_count > 1")
	snapshot, err := Compile(config)
	require.NoError(t, err)
	config.Events[0].Strategies[0].Action.ActionParams["options"].([]any)[0].(map[string]any)["level_name"] = "mutated"
	for _, count := range []int64{2, 1, 2} {
		result, err := snapshot.Execute(context.Background(), RiskInput{UniqueID: "test", EventCode: "PRODUCT", ObjectType: "CUSTOMER", Param: map[string]any{"count": count}})
		require.NoError(t, err)
		assert.Contains(t, result.FactorResults, "f_count", "factor referenced only by option condition must execute")
		options := result.Actions[0].ActionParams["options"].([]any)
		if count > 1 {
			require.Len(t, options, 2)
			assert.Equal(t, "conditional", options[0].(map[string]any)["level_name"])
		} else {
			require.Len(t, options, 1)
		}
		for _, o := range options {
			assert.NotContains(t, o.(map[string]any), "condition")
		}
		options[0].(map[string]any)["level_name"] = "client mutation"
	}
	result, err := snapshot.Execute(context.Background(), RiskInput{UniqueID: "test", EventCode: "PRODUCT", ObjectType: "CUSTOMER", Param: map[string]any{"count": risk.ParamValue{Error: true}}})
	require.Error(t, err)
	assert.Empty(t, result.DecisionCode)
	assert.Empty(t, result.Actions)
}

func TestConditionalOptionsRejectInvalidConfiguration(t *testing.T) {
	for _, condition := range []any{"", 123, "f_unknown > 0", "f_count"} {
		t.Run("invalid condition", func(t *testing.T) { _, err := Compile(optionConfig(condition)); assert.Error(t, err) })
	}
}

func TestOptionConditionFailureIsNotBusinessDecision(t *testing.T) {
	config := optionConfig("safe_divide(f_count, 0) > 0")
	snapshot, err := Compile(config)
	require.NoError(t, err)
	result, err := snapshot.Execute(context.Background(), RiskInput{UniqueID: "test", EventCode: "PRODUCT", ObjectType: "CUSTOMER", Param: map[string]any{"count": int64(1)}})
	require.Error(t, err)
	assert.Empty(t, result.DecisionCode)
	assert.Empty(t, result.Actions)
	assert.Equal(t, StrategyStatusExecuteFailed, result.StrategyResults[0].Status)
}

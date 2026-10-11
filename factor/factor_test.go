package factor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wilebeast/decision-engine/compiler"
	"github.com/wilebeast/decision-engine/model"
)

func TestRegistryNewFactor(t *testing.T) {
	factor, err := BuiltinRegistry().NewFactor(risk.FactorDefinition{Code: "f_count", Type: risk.FactorTypeValue, ParamKey: "count", ValueType: risk.ValueTypeInt})

	assert.NoError(t, err)
	assert.Equal(t, "f_count", factor.Code())
	result := factor.Fetch(context.Background(), testRuntime{param: map[string]any{"count": float64(12)}})
	assert.Equal(t, risk.FactorStatusSuccess, result.Status)
	assert.Equal(t, int64(12), result.Value)
}

func TestBuiltinRegistryIsInitialized(t *testing.T) {
	assert.NotNil(t, BuiltinRegistry())
}

func TestRegistryRejectsDuplicateAndUnsupportedProviderTypes(t *testing.T) {
	registry := NewRegistry()
	assert.NoError(t, registry.RegisterProvider(valueProvider{}))
	assert.Error(t, registry.RegisterProvider(valueProvider{}))

	_, err := BuiltinRegistry().NewFactor(risk.FactorDefinition{Code: "f_unknown", Type: risk.FactorType("UNKNOWN")})
	assert.Error(t, err)
}

func TestExpressionFactorFetchReturnsDependencyError(t *testing.T) {
	factor := &ExpressionFactor{factorCode: "f_rate", valueType: risk.ValueTypeFloat, directDependentFactorCodes: []string{"f_count"}}
	result := factor.Fetch(context.Background(), testRuntime{})
	assert.Equal(t, risk.FactorStatusError, result.Status)
	assert.Equal(t, "DEPENDENCY_ERROR", result.ErrorCode)
}

func TestRegistryNewFactorCompilesExpressionAndDependencies(t *testing.T) {
	closeCompiler := compiler.NewCompiler(map[string]any{"f_count": int64(0)})
	defer closeCompiler()
	factor, err := BuiltinRegistry().NewFactor(risk.FactorDefinition{Code: "f_ratio", Type: risk.FactorTypeExpression, Expression: "safe_divide(f_count, f_count)", ValueType: risk.ValueTypeFloat})

	assert.NoError(t, err)
	assert.Equal(t, []string{"f_count"}, factor.DependentFactorCodes())
	result := factor.Fetch(context.Background(), testRuntime{factors: map[string]risk.FactorResult{"f_count": {Code: "f_count", Status: risk.FactorStatusSuccess, Value: int64(2)}}})
	assert.Equal(t, risk.FactorStatusSuccess, result.Status)
	assert.Equal(t, float64(1), result.Value)
}

type testRuntime struct {
	param   map[string]any
	factors map[string]risk.FactorResult
}

func (r testRuntime) ParamValue(key string) (any, bool) {
	value, ok := r.param[key]
	return value, ok
}

func (r testRuntime) GetFactor(code string) (risk.FactorResult, bool) {
	result, ok := r.factors[code]
	return result, ok
}

package factor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wilebeast/decision-engine/model"
	"testing"
	"time"
)

func TestValueTypesAndMissing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		typ    risk.ValueType
		value  any
		status risk.FactorStatus
	}{
		{"false", risk.ValueTypeBool, false, risk.FactorStatusSuccess},
		{"string", risk.ValueTypeString, "approved", risk.FactorStatusSuccess},
		{"strings", risk.ValueTypeStrings, []string{"GB"}, risk.FactorStatusSuccess},
		{"date", risk.ValueTypeDate, time.Now(), risk.FactorStatusSuccess},
		{"missing", risk.ValueTypeBool, risk.ParamValue{Missing: true}, risk.FactorStatusMissing},
		{"source failure", risk.ValueTypeBool, risk.ParamValue{Error: true}, risk.FactorStatusError},
		{"wrong type", risk.ValueTypeBool, "false", risk.FactorStatusError},
		{"fraction", risk.ValueTypeInt, 1.1, risk.FactorStatusError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := BuiltinRegistry().NewFactor(risk.FactorDefinition{Code: "f_value", Type: risk.FactorTypeValue, ParamKey: "value", ValueType: tc.typ})
			require.NoError(t, err)
			v := f.Fetch(context.Background(), testRuntime{param: map[string]any{"value": tc.value}})
			assert.Equal(t, tc.status, v.Status)
		})
	}
}

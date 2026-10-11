package factor

import (
	"context"
	"math"
	"time"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/model"
)

type ValueFactor struct {
	code      string
	paramKey  string
	valueType risk.ValueType
}

func init() {
	if err := builtinRegistry.RegisterProvider(valueProvider{}); err != nil {
		panic(err)
	}
}

func (f *ValueFactor) Code() string                   { return f.code }
func (f *ValueFactor) ParamKey() string               { return f.paramKey }
func (f *ValueFactor) DependentFactorCodes() []string { return nil }
func (f *ValueFactor) Fetch(_ context.Context, runtime Runtime) risk.FactorResult {
	value, ok := runtime.ParamValue(f.paramKey)
	if !ok {
		return errorResult(f.Code(), "MISSING_PARAM", "required param is missing")
	}
	if fact, wrapped := value.(risk.ParamValue); wrapped {
		if fact.Error {
			return errorResult(f.Code(), "SOURCE_ERROR", "required fact could not be read")
		}
		if fact.Missing {
			return risk.FactorResult{Code: f.Code(), Status: risk.FactorStatusMissing}
		}
		value = fact.Value
	}
	converted, err := convertValue(value, f.valueType)
	if err != nil {
		return errorResult(f.Code(), "TYPE_MISMATCH", err.Error())
	}
	return risk.FactorResult{Code: f.Code(), Status: risk.FactorStatusSuccess, Value: converted}
}

type valueProvider struct{}

func (valueProvider) Type() risk.FactorType { return risk.FactorTypeValue }
func (valueProvider) NewFactor(def risk.FactorDefinition) (Factor, error) {
	if def.ParamKey == "" || def.Expression != "" {
		return nil, gerror.NewCodef(gcode.CodeInvalidParameter, "VALUE factor %s must define only param key", def.Code)
	}
	if _, err := ZeroValue(def.ValueType); err != nil {
		return nil, err
	}
	return &ValueFactor{code: def.Code, paramKey: def.ParamKey, valueType: def.ValueType}, nil
}

func ZeroValue(valueType risk.ValueType) (any, error) {
	switch valueType {
	case risk.ValueTypeInt:
		return int64(0), nil
	case risk.ValueTypeFloat:
		return float64(0), nil
	case risk.ValueTypeBool:
		return false, nil
	case risk.ValueTypeString:
		return "", nil
	case risk.ValueTypeStrings:
		return []string{}, nil
	case risk.ValueTypeDate:
		return time.Time{}, nil
	case risk.ValueTypeObject:
		return nil, nil
	default:
		return nil, gerror.NewCodef(gcode.CodeInvalidParameter, "unsupported factor value type: %s", valueType)
	}
}

func convertValue(value any, valueType risk.ValueType) (any, error) {
	switch valueType {
	case risk.ValueTypeBool:
		if v, ok := value.(bool); ok {
			return v, nil
		}
	case risk.ValueTypeString:
		if v, ok := value.(string); ok {
			return v, nil
		}
	case risk.ValueTypeStrings:
		if v, ok := value.([]string); ok {
			return v, nil
		}
		if values, ok := value.([]any); ok {
			result := make([]string, len(values))
			for i, item := range values {
				text, ok := item.(string)
				if !ok {
					return nil, gerror.NewCode(gcode.CodeInvalidParameter, "string collection contains a non-string")
				}
				result[i] = text
			}
			return result, nil
		}
	case risk.ValueTypeDate:
		if v, ok := value.(time.Time); ok && !v.IsZero() {
			return v, nil
		}
	case risk.ValueTypeObject:
		if value != nil {
			return value, nil
		}
	case risk.ValueTypeInt:
		switch typed := value.(type) {
		case int64:
			return typed, nil
		case int:
			return int64(typed), nil
		case float64:
			if math.IsNaN(typed) || math.IsInf(typed, 0) || typed >= math.Exp2(63) || typed < -math.Exp2(63) || math.Trunc(typed) != typed {
				return nil, gerror.NewCode(gcode.CodeInvalidParameter, "integer factor has a fractional value")
			}
			return int64(typed), nil
		}
	case risk.ValueTypeFloat:
		switch typed := value.(type) {
		case float64:
			return typed, nil
		case int64:
			return float64(typed), nil
		}
	}
	return nil, gerror.NewCode(gcode.CodeInvalidParameter, "factor value does not match value type")
}

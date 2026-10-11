package factor

import (
	"context"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/compiler"
	"github.com/wilebeast/decision-engine/model"
)

type ExpressionFactor struct {
	factorCode                 string
	valueType                  risk.ValueType
	directDependentFactorCodes []string
	program                    *vm.Program
	functions                  map[string]any
}

func init() {
	if err := builtinRegistry.RegisterProvider(expressionProvider{}); err != nil {
		panic(err)
	}
}

func (f *ExpressionFactor) Code() string                   { return f.factorCode }
func (f *ExpressionFactor) DependentFactorCodes() []string { return f.directDependentFactorCodes }
func (f *ExpressionFactor) Fetch(_ context.Context, runtime Runtime) risk.FactorResult {
	for _, code := range f.directDependentFactorCodes {
		if result, ok := runtime.GetFactor(code); !ok || (result.Status != risk.FactorStatusSuccess && result.Status != risk.FactorStatusMissing) {
			return errorResult(f.Code(), "DEPENDENCY_ERROR", "dependency factor is unavailable")
		}
	}
	env := make(map[string]any, len(f.directDependentFactorCodes))
	for key, fn := range f.functions {
		env[key] = fn
	}
	for _, code := range f.directDependentFactorCodes {
		result, _ := runtime.GetFactor(code)
		env[code] = result.Value
	}
	value, err := expr.Run(f.program, env)
	if err != nil {
		return errorResult(f.Code(), "EXPRESSION_EVALUATION_ERROR", "expression execution failed")
	}
	converted, err := convertValue(value, f.valueType)
	if err != nil {
		return errorResult(f.Code(), "RESULT_TYPE_MISMATCH", err.Error())
	}
	return risk.FactorResult{Code: f.Code(), Status: risk.FactorStatusSuccess, Value: converted}
}

type expressionProvider struct{}

func (expressionProvider) Type() risk.FactorType { return risk.FactorTypeExpression }
func (expressionProvider) NewFactor(def risk.FactorDefinition) (Factor, error) {
	if def.Expression == "" || def.ParamKey != "" {
		return nil, gerror.NewCodef(gcode.CodeInvalidParameter, "EXPRESSION factor %s must define only expression", def.Code)
	}
	if _, err := ZeroValue(def.ValueType); err != nil {
		return nil, err
	}
	compiledExpression, err := compiler.CompileExpression(def.Expression)
	if err != nil {
		return nil, gerror.WrapCodef(gcode.CodeInvalidParameter, err, "compile expression factor %s", def.Code)
	}
	return &ExpressionFactor{factorCode: def.Code, valueType: def.ValueType, directDependentFactorCodes: compiledExpression.DirectDependentFactorCodes, program: compiledExpression.Program, functions: compiledExpression.Functions}, nil
}

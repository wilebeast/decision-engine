package compiler

import (
	"maps"
	"regexp"
	"sort"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

type CompiledExpression struct {
	DirectDependentFactorCodes []string
	Program                    *vm.Program
	Functions                  map[string]any
}

type compiler struct {
	factorTypes map[string]any
	functions   map[string]any
}

var (
	currentCompiler  *compiler
	compileMu        sync.Mutex
	factorCodeRegexp = regexp.MustCompile(`\bf_[A-Za-z0-9_]+\b`)
)

// NewCompiler starts one global compile session. It must be called before a
// factor or rule compiles an expression, and the returned function must be
// deferred by the caller. Runtime evaluation never uses this compiler.
func NewCompiler(factorTypes map[string]any, functions ...map[string]any) func() {
	compileMu.Lock()
	currentCompiler = &compiler{factorTypes: maps.Clone(factorTypes), functions: map[string]any{}}
	if len(functions) > 0 {
		currentCompiler.functions = maps.Clone(functions[0])
	}
	return func() {
		currentCompiler = nil
		compileMu.Unlock()
	}
}

func CompileExpression(expression string) (CompiledExpression, error) {
	return compile(expression)
}

func CompileRule(expression string) (CompiledExpression, error) {
	return compile(expression, expr.AsBool())
}

func compile(expression string, options ...expr.Option) (CompiledExpression, error) {
	if currentCompiler == nil {
		return CompiledExpression{}, gerror.NewCode(gcode.CodeInvalidOperation, "expression compiler is unavailable outside a compile session")
	}
	directDependentFactorCodes := extractDirectDependentFactorCodes(expression)
	env := make(map[string]any, len(directDependentFactorCodes))
	for name, fn := range currentCompiler.functions {
		env[name] = fn
	}
	for _, factorCode := range directDependentFactorCodes {
		factorType, ok := currentCompiler.factorTypes[factorCode]
		if !ok {
			return CompiledExpression{}, gerror.NewCodef(gcode.CodeNotFound, "unknown factor type: %s", factorCode)
		}
		env[factorCode] = factorType
	}
	options = append([]expr.Option{expr.Env(env)}, options...)
	options = append(options, expr.Function("safe_divide", safeDivide, new(func(int64, int64) (float64, error))))
	program, err := expr.Compile(expression, options...)
	if err != nil {
		return CompiledExpression{}, err
	}
	return CompiledExpression{DirectDependentFactorCodes: directDependentFactorCodes, Program: program, Functions: maps.Clone(currentCompiler.functions)}, nil
}

func extractDirectDependentFactorCodes(expression string) []string {
	matches := factorCodeRegexp.FindAllString(expression, -1)
	unique := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		unique[match] = struct{}{}
	}
	codes := make([]string, 0, len(unique))
	for code := range unique {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func safeDivide(params ...any) (any, error) {
	if len(params) != 2 {
		return nil, gerror.NewCode(gcode.CodeInvalidParameter, "safe_divide requires two arguments")
	}
	numerator, numeratorOK := params[0].(int64)
	denominator, denominatorOK := params[1].(int64)
	if !numeratorOK || !denominatorOK {
		return nil, gerror.NewCode(gcode.CodeInvalidParameter, "safe_divide only supports int64 values")
	}
	if denominator <= 0 {
		return nil, gerror.NewCode(gcode.CodeInvalidParameter, "division by zero")
	}
	return float64(numerator) / float64(denominator), nil
}

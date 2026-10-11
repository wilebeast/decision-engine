package compiler

import (
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompileContract(t *testing.T) {
	_, err := CompileRule("true")
	assert.Error(t, err)
	close := NewCompiler(map[string]any{"f_n": int64(0)})
	defer close()
	p, err := CompileRule("f_n > 1")
	require.NoError(t, err)
	assert.Equal(t, []string{"f_n"}, p.DirectDependentFactorCodes)
	result, err := expr.Run(p.Program, map[string]any{"f_n": int64(2)})
	require.NoError(t, err)
	assert.Equal(t, true, result)
	_, err = CompileRule("f_unknown > 1")
	assert.Error(t, err)
	_, err = CompileRule("\"text\"")
	assert.Error(t, err)
	p, err = CompileExpression("safe_divide(f_n, f_n)")
	require.NoError(t, err)
	_, err = expr.Run(p.Program, map[string]any{"f_n": int64(0)})
	assert.Error(t, err)
}

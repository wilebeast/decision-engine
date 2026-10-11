package factor

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/model"
)

// Runtime provides the data a Factor may read during one event evaluation.
// The risk engine owns its implementation and lifecycle.
type Runtime interface {
	ParamValue(key string) (any, bool)
	GetFactor(code string) (risk.FactorResult, bool)
}

type Factor interface {
	Code() string
	DependentFactorCodes() []string
	Fetch(ctx context.Context, runtime Runtime) risk.FactorResult
}

// Provider creates a runtime Factor for one configured factor type.
// It is intentionally distinct from Factor, which is used during evaluation.
type Provider interface {
	Type() risk.FactorType
	NewFactor(risk.FactorDefinition) (Factor, error)
}

type Registry struct {
	providers map[risk.FactorType]Provider
}

var builtinRegistry = NewRegistry()

// BuiltinRegistry returns the static registry of factor types supported by this service.
func BuiltinRegistry() *Registry { return builtinRegistry }

func NewRegistry() *Registry { return &Registry{providers: make(map[risk.FactorType]Provider)} }
func (r *Registry) RegisterProvider(provider Provider) error {
	if _, ok := r.providers[provider.Type()]; ok {
		return gerror.NewCodef(gcode.CodeInvalidParameter, "duplicate factor provider: %s", provider.Type())
	}
	r.providers[provider.Type()] = provider
	return nil
}

func (r *Registry) NewFactor(def risk.FactorDefinition) (Factor, error) {
	provider, ok := r.providers[def.Type]
	if !ok {
		return nil, gerror.NewCodef(gcode.CodeNotSupported, "unsupported factor type: %s", def.Type)
	}
	return provider.NewFactor(def)
}

func errorResult(code, errorCode, message string) risk.FactorResult {
	return risk.FactorResult{Code: code, Status: risk.FactorStatusError, ErrorCode: errorCode, ErrorMessage: message}
}

package engine

import (
	"maps"

	"github.com/wilebeast/decision-engine/model"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/wilebeast/decision-engine/factor"
)

type factorSlot struct {
	result    FactorResult
	completed bool
}

type strategySlot struct {
	result    StrategyResult
	completed bool
}

// RuntimeState holds all mutable data for one event evaluation.
// It must never be shared between requests.
type RuntimeState struct {
	inputParam    map[string]any
	paramReader   func(string) (any, bool)
	factorValues  map[string]any
	factorSlots   map[string]*factorSlot
	strategySlots []strategySlot
}

func newRuntimeState(inputParam map[string]any, factors []factor.Factor, strategies []compiledStrategy) *RuntimeState {
	factorSlots := make(map[string]*factorSlot, len(factors))
	for _, item := range factors {
		factorSlots[item.Code()] = &factorSlot{}
	}
	strategySlots := make([]strategySlot, len(strategies))
	for index, strategy := range strategies {
		strategySlots[index].result = StrategyResult{StrategyCode: strategy.strategyCode, DecisionCode: strategy.decisionCode, RuleCode: strategy.rule.ruleCode, Status: StrategyStatusPending}
	}
	return &RuntimeState{
		inputParam:    maps.Clone(inputParam),
		factorValues:  make(map[string]any, len(factors)),
		factorSlots:   factorSlots,
		strategySlots: strategySlots,
	}
}

func (s *RuntimeState) ParamValue(key string) (any, bool) {
	if s.paramReader != nil {
		return s.paramReader(key)
	}
	value, ok := s.inputParam[key]
	return value, ok
}

func (s *RuntimeState) GetFactor(code string) (FactorResult, bool) {
	slot, ok := s.factorSlots[code]
	if !ok || !slot.completed {
		return FactorResult{}, false
	}
	return slot.result, true
}

func (s *RuntimeState) CompleteFactor(code string, result FactorResult) error {
	slot, ok := s.factorSlots[code]
	if !ok {
		return gerror.NewCodef(gcode.CodeNotFound, "unknown factor: %s", code)
	}
	if slot.completed {
		return gerror.NewCodef(gcode.CodeInvalidOperation, "factor already completed: %s", code)
	}
	slot.result = result
	slot.completed = true
	return nil
}

func (s *RuntimeState) FactorSnapshot() map[string]FactorResult {
	results := make(map[string]FactorResult, len(s.factorSlots))
	for code, slot := range s.factorSlots {
		if slot.completed {
			results[code] = slot.result
		}
	}
	return results
}

func (s *RuntimeState) MaterializeFactorValues() {
	for code, result := range s.FactorSnapshot() {
		if result.Status == FactorStatusSuccess {
			s.factorValues[code] = result.Value
		} else if result.Status == risk.FactorStatusMissing {
			s.factorValues[code] = nil
		}
	}
}

func (s *RuntimeState) FactorValues() map[string]any { return s.factorValues }

func (s *RuntimeState) StrategyResult(index int) (StrategyResult, error) {
	if index < 0 || index >= len(s.strategySlots) {
		return StrategyResult{}, gerror.NewCodef(gcode.CodeInvalidParameter, "strategy index out of range: %d", index)
	}
	return s.strategySlots[index].result, nil
}

func (s *RuntimeState) CompleteStrategy(index int, result StrategyResult) error {
	if index < 0 || index >= len(s.strategySlots) {
		return gerror.NewCodef(gcode.CodeInvalidParameter, "strategy index out of range: %d", index)
	}
	if s.strategySlots[index].completed {
		return gerror.NewCodef(gcode.CodeInvalidOperation, "strategy already completed at index: %d", index)
	}
	s.strategySlots[index].result = result
	s.strategySlots[index].completed = true
	return nil
}

func (s *RuntimeState) StrategySnapshot() []StrategyResult {
	results := make([]StrategyResult, len(s.strategySlots))
	for index, slot := range s.strategySlots {
		results[index] = slot.result
	}
	return results
}

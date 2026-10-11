package risk

import "time"

type RiskInput struct {
	UniqueID   string         `json:"unique_id"`
	EventCode  string         `json:"event_code"`
	ObjectType string         `json:"object_type"`
	ObjectID   string         `json:"object_id"`
	Param      map[string]any `json:"param"`
	// ParamReader adapts EvaluationContext's key -> getter map. Not client input.
	ParamReader func(string) (any, bool) `json:"-"`
}

type FactorStatus string

const (
	FactorStatusSuccess FactorStatus = "SUCCESS"
	FactorStatusError   FactorStatus = "ERROR"
	FactorStatusMissing FactorStatus = "MISSING"
)

type FactorResult struct {
	Code         string       `json:"code"`
	Status       FactorStatus `json:"status"`
	Value        any          `json:"value,omitempty"`
	ErrorCode    string       `json:"error_code,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

type StrategyStatus string

const (
	StrategyStatusPending             StrategyStatus = "PENDING"
	StrategyStatusConditionNotMatched StrategyStatus = "CONDITION_NOT_MATCHED"
	StrategyStatusHit                 StrategyStatus = "HIT"
	StrategyStatusInputInvalid        StrategyStatus = "INPUT_INVALID"
	StrategyStatusExecuteFailed       StrategyStatus = "EXECUTE_FAILED"
)

type StrategyResult struct {
	StrategyCode string         `json:"strategy_code"`
	DecisionCode string         `json:"decision_code"`
	RuleCode     string         `json:"rule_code"`
	IsHit        bool           `json:"is_hit"`
	Status       StrategyStatus `json:"status"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Duration     time.Duration  `json:"duration"`
}

type DecisionResult struct {
	UniqueID     string  `json:"unique_id"`
	EventCode    string  `json:"event_code"`
	Subject      Subject `json:"subject"`
	StrategyCode string  `json:"strategy_code,omitempty"`
	RuleCode     string  `json:"rule_code,omitempty"`
	DecisionCode string  `json:"decision_code,omitempty"`
	Priority     int     `json:"priority,omitempty"`
	Action       *Action `json:"action,omitempty"`
	// The engine aggregates all hit strategies instead of choosing a single winner.
	Actions         []Action                `json:"actions"`
	FactorResults   map[string]FactorResult `json:"factor_results"`
	StrategyResults []StrategyResult        `json:"strategy_results"`
}

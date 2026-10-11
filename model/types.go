package risk

import "time"

type Subject struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
}

type Action struct {
	ActionCode   string         `json:"action_code"`
	ActionParams map[string]any `json:"action_params"`
}

type ActionCommand struct {
	UniqueID     string    `json:"unique_id"`
	EventCode    string    `json:"event_code,omitempty"`
	Subject      Subject   `json:"subject"`
	RuleCode     string    `json:"rule_code,omitempty"`
	DecisionCode string    `json:"decision_code,omitempty"`
	Action       Action    `json:"action"`
	SourceType   string    `json:"source_type"`
	SourceRef    string    `json:"source_ref"`
	OperatorID   string    `json:"operator_id"`
	Reason       string    `json:"reason,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
}

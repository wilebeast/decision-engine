package risk

type ValueType string

const (
	ValueTypeInt     ValueType = "INT"
	ValueTypeFloat   ValueType = "FLOAT"
	ValueTypeBool    ValueType = "BOOL"
	ValueTypeString  ValueType = "STRING"
	ValueTypeStrings ValueType = "STRINGS"
	ValueTypeDate    ValueType = "DATE"
	// Structured facts are validated by their registered context reader.
	ValueTypeObject ValueType = "OBJECT"
)

// ParamValue distinguishes a successful read with no data from a failed read.
// The engine never obtains external facts itself.
type ParamValue struct {
	Value   any
	Missing bool
	Error   bool
}

type FactorType string

const (
	FactorTypeValue      FactorType = "VALUE"
	FactorTypeExpression FactorType = "EXPRESSION"
)

type FactorDefinition struct {
	Code       string
	Type       FactorType
	ParamKey   string
	Expression string
	ValueType  ValueType
}

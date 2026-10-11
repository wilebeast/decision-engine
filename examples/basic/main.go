package main

import (
	"context"
	"fmt"
	"log"

	engine "github.com/wilebeast/decision-engine"
	risk "github.com/wilebeast/decision-engine/model"
)

func main() {
	snapshot, err := engine.Compile(engine.Config{
		Factors: []engine.FactorDefinition{{Code: "f_count", Type: engine.FactorTypeValue, ValueType: engine.ValueTypeInt, ParamKey: "count"}},
		Events: []engine.EventDefinition{{EventCode: "demo", Strategies: []engine.StrategyDefinition{{
			StrategyCode: "capacity", Rule: engine.RuleDefinition{RuleCode: "over_limit", Expression: "f_count > 10"},
			Decision: engine.DecisionDefinition{DecisionCode: "continue", Cases: []engine.DecisionCase{{Condition: "f_count > 100", DecisionCode: "block"}}},
			Action:   risk.Action{ActionCode: "request_review"},
		}}}},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, count := range []int64{5, 20, 200} {
		result, err := snapshot.Execute(context.Background(), engine.RiskInput{
			UniqueID: fmt.Sprintf("demo-%d", count), EventCode: "demo", ObjectType: "example", ObjectID: "sample",
			Param: map[string]any{"count": count},
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("count=%d decision=%s actions=%d\n", count, result.DecisionCode, len(result.Actions))
	}
}

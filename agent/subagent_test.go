package agent

import (
	"testing"

	"github.com/ParthPant/pathfinder/tools"
)

func TestSpawnSubagentToolDefinition(t *testing.T) {
	if SpawnSubagentTool.Name != "spawn_subagent" {
		t.Errorf("expected name 'spawn_subagent', got '%s'", SpawnSubagentTool.Name)
	}
	if SpawnSubagentTool.Type != "function" {
		t.Errorf("expected type 'function', got '%s'", SpawnSubagentTool.Type)
	}

	params := tools.ParamsFor[SpawnSubagentInput]()
	requiredFields := []string{"task"}
	for _, field := range requiredFields {
		if _, ok := params.Properties[field]; !ok {
			t.Errorf("expected '%s' property in SpawnSubagentInput", field)
		}
	}

	for _, req := range params.Required {
		if req == "task" {
			return // task must be required, test passes
		}
	}
	t.Error("expected 'task' to be a required field")
}

func TestSpawnSubagentInputOptionalFields(t *testing.T) {
	params := tools.ParamsFor[SpawnSubagentInput]()
	optionalFields := []string{"files", "reasoning", "available_tools"}
	for _, field := range optionalFields {
		if _, ok := params.Properties[field]; !ok {
			t.Errorf("expected '%s' property in SpawnSubagentInput", field)
		}
	}
}

func TestSpawnSubagentInputReasoningEnum(t *testing.T) {
	params := tools.ParamsFor[SpawnSubagentInput]()
	reasoning, ok := params.Properties["reasoning"]
	if !ok {
		t.Fatal("expected 'reasoning' property")
	}
	if len(reasoning.Enum) == 0 {
		t.Error("expected 'reasoning' to have enum values")
	}
}

func TestSpawnSubagentInputAvailableToolsEnum(t *testing.T) {
	params := tools.ParamsFor[SpawnSubagentInput]()
	toolsProp, ok := params.Properties["available_tools"]
	if !ok {
		t.Fatal("expected 'available_tools' property")
	}
	if len(toolsProp.Enum) == 0 {
		t.Error("expected 'available_tools' to have enum values")
	}
}
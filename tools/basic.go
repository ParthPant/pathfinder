package tools

import (
	"context"
	"time"
)

// Date time tool

var GetDateTimeParams = ToolParams{
	Properties: map[string]ParamProperty{},
}

var GetDateTimeTool = FunctionDefinition{
	Type:        "function",
	Name:        "get_date_time",
	Description: "Use this tool to get the current date and time.",
	Parameters:  ParamsFor[struct{}](),
	Strict:      false,
	Function:    GetDateTime,
}

func GetDateTime(ctx context.Context, params struct{}) (any, error) {
	return time.Now().String(), nil
}

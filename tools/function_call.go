package tools

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
)

type ParamProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitzero"`
	Enum        []string `json:"enum,omitzero"`
}

type ToolParams struct {
	Properties           map[string]ParamProperty `json:"properties"`
	Required             []string                 `json:"required,omitzero"`
	AdditionalProperties bool                     `json:"additional_properties,omitzero"`
}

type FunctionDefinition struct {
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitzero"`
	Parameters  ToolParams `json:"params"`
	Strict      bool       `json:"strict"`
	Function    any        `json:"-"`
}

func NewFunctionDefinition(name string,
	desc string,
	parameters ToolParams,
	strict bool,
	function any,
) (FunctionDefinition, error) {

	err := validateFunctionSignature(function)
	if err != nil {
		return FunctionDefinition{}, err
	}

	return FunctionDefinition{
		"function",
		name,
		desc,
		parameters,
		strict,
		function,
	}, nil
}

func validateFunctionSignature(fn any) error {
	v := reflect.ValueOf(fn)

	if v.Kind() != reflect.Func {
		return errors.New("Provided object is not a function.")
	}

	ctxType := reflect.TypeFor[context.Context]()
	errorType := reflect.TypeFor[error]()

	t := v.Type()
	if t.NumIn() == 0 || !t.In(0).Implements(ctxType) {
		return errors.New("Funciton should accept context.Context as first parameter.")
	}

	if t.NumIn() != 2 {
		return errors.New("Function should accept only two parameters, first a context.Context and send a Schema struct.")
	}

	if t.NumOut() != 2 || !t.Out(1).Implements(errorType) {
		return errors.New("Function should return (any, error)")
	}
	return nil
}

var paramTypes = map[string]string{
	"string": "string",
	"int":    "integer",
	"int32":  "integer",
	"int64":  "integer",
	"bool":   "boolean",
}

func ParamsFor[T any]() ToolParams {
	properties := make(map[string]ParamProperty)
	requiredFields := make([]string, 0)

	t := reflect.TypeFor[T]()
	for field := range t.Fields() {
		name, desc, required := readTag(&field)
		properties[name] = ParamProperty{
			Type:        paramTypes[field.Type.Name()],
			Description: desc,
		}
		if required {
			requiredFields = append(requiredFields, name)
		}
	}
	slog.Debug("Generated ToolParams", "Properties", properties, "Required", requiredFields)
	return ToolParams{
		Properties: properties,
		Required:   requiredFields,
	}
}

func readTag(f *reflect.StructField) (string, string, bool) {
	jsonTag, _ := f.Tag.Lookup("json")

	var name string = strings.Split(jsonTag, ",")[0]
	var desc string = ""
	var required bool = false

	toolTag, _ := f.Tag.Lookup("tool")
	splits := strings.Split(toolTag, ",")

	if len(splits) >= 1 {
		desc = strings.Trim(splits[0], " ")
	}
	if len(splits) >= 2 {
		required = strings.Trim(splits[1], " ") == "required"
	}

	return name, desc, required
}

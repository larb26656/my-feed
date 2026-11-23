package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"

	api "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

type Registry struct {
	definitions []api.Definition
	schemas     map[string]*jsonschema.Schema
	handlers    map[string]api.Definition
}

func New(definitions ...api.Definition) (*Registry, error) {
	r := &Registry{schemas: make(map[string]*jsonschema.Schema), handlers: make(map[string]api.Definition)}
	for _, d := range definitions {
		if err := validateDefinition(d); err != nil {
			return nil, fmt.Errorf("invalid source type %q: %w", d.Code, err)
		}
		if _, exists := r.schemas[d.Code]; exists {
			return nil, fmt.Errorf("duplicate source type %q", d.Code)
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("schema.json", d.Schema); err != nil {
			return nil, fmt.Errorf("source type %s: %w", d.Code, err)
		}
		schema, err := c.Compile("schema.json")
		if err != nil {
			return nil, fmt.Errorf("source type %s: %w", d.Code, err)
		}
		r.schemas[d.Code] = schema
		r.handlers[d.Code] = d
		r.definitions = append(r.definitions, d)
	}
	return r, nil
}

func (r *Registry) Definitions() []api.Definition {
	return append([]api.Definition(nil), r.definitions...)
}

// Prepare validates the published input before applying type-specific normalization.
func (r *Registry) Prepare(ctx context.Context, code string, config map[string]any) (map[string]any, string, error) {
	schema, ok := r.schemas[code]
	if !ok {
		return nil, "", apperr.InvalidInput.New("unknown source type").WithDetail(map[string]any{"field": "type"})
	}
	if err := schema.Validate(config); err != nil {
		return nil, "", apperr.InvalidInput.Wrap(err, "invalid source config").WithDetail(map[string]any{"fields": fieldErrors(err)})
	}
	if normalize := r.handlers[code].Normalize; normalize != nil {
		return normalize(ctx, config)
	}
	return nil, "", apperr.InvalidInput.New("unsupported source type")
}

func fieldErrors(err error) map[string]string {
	fields := map[string]string{}
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return fields
	}
	var visit func(*jsonschema.ValidationError)
	visit = func(e *jsonschema.ValidationError) {
		if required, ok := e.ErrorKind.(*kind.Required); ok {
			for _, name := range required.Missing {
				fields[name] = "required"
			}
		}
		if additional, ok := e.ErrorKind.(*kind.AdditionalProperties); ok {
			for _, name := range additional.Properties {
				fields[name] = "unknown property"
			}
		}
		if len(e.InstanceLocation) > 0 && len(e.Causes) == 0 {
			fields[strings.Join(e.InstanceLocation, ".")] = "invalid value"
		}
		for _, cause := range e.Causes {
			visit(cause)
		}
	}
	visit(validation)
	return fields
}

func validateDefinition(d api.Definition) error {
	if d.Normalize == nil || d.Obtainer == nil {
		return fmt.Errorf("normalization and obtain handlers are required")
	}
	if d.Code == "" || d.Code == "manual" || strings.TrimSpace(d.Label) == "" {
		return fmt.Errorf("code or label is invalid")
	}
	draft, _ := d.Schema["$schema"].(string)
	objectType, _ := d.Schema["type"].(string)
	additionalProperties, closed := d.Schema["additionalProperties"].(bool)
	if draft != "https://json-schema.org/draft/2020-12/schema" || objectType != "object" || !closed || additionalProperties {
		return fmt.Errorf("schema must describe a closed Draft 2020-12 object")
	}
	for key := range d.Schema {
		switch key {
		case "$schema", "type", "additionalProperties", "properties", "required", "title", "description", "allOf":
		default:
			return fmt.Errorf("unsupported schema keyword %s", key)
		}
	}
	properties, ok := d.Schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 || len(d.UI) != len(properties) {
		return fmt.Errorf("fields and UI metadata must agree")
	}
	for name, raw := range properties {
		field, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("field %s has unsupported schema", name)
		}
		for key := range field {
			switch key {
			case "type", "enum", "pattern", "minimum", "maximum", "minLength", "maxLength", "default", "title", "description":
			default:
				return fmt.Errorf("field %s has unsupported schema keyword %s", name, key)
			}
		}
		fieldType, _ := field["type"].(string)
		ui, ok := d.UI[name].(map[string]any)
		if !ok {
			return fmt.Errorf("field %s has no UI metadata", name)
		}
		for key := range ui {
			switch key {
			case "label", "widget", "description", "error":
			default:
				return fmt.Errorf("field %s has unsupported UI metadata %s", name, key)
			}
		}
		label, ok := ui["label"].(string)
		if !ok || strings.TrimSpace(label) == "" {
			return fmt.Errorf("field %s has no label", name)
		}
		widget, _ := ui["widget"].(string)

		valid := false
		switch widget {
		case "text", "url":
			valid = fieldType == "string"
		case "number":
			valid = fieldType == "number" || fieldType == "integer"
		case "checkbox":
			valid = fieldType == "boolean"
		case "select":
			_, hasEnum := field["enum"]
			valid = hasEnum && (fieldType == "string" || fieldType == "number" || fieldType == "integer")
		}
		if !valid {
			return fmt.Errorf("field %s has an unsupported widget or type", name)
		}
	}
	return nil
}

// Obtain dispatches both Preview and Fetch Job requests to the registered handler.
func (r *Registry) Obtain(ctx context.Context, in *api.Input) ([]api.PreparedEntry, error) {
	if in == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	handler, ok := r.handlers[string(in.Type)]
	if !ok || handler.Obtainer == nil {
		return nil, apperr.ResourceConflict.New("no obtain handler registered")
	}
	return handler.Obtainer.Obtain(ctx, in)
}

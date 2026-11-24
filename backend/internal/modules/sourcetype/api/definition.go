package api

import "context"

// Definition describes accepted input and its presentation to readers.
type Definition struct {
	Normalize  func(context.Context, map[string]any) (map[string]any, string, error) `json:"-"`
	Obtainer   Obtainer                                                              `json:"-"`
	Code       string                                                                `json:"code"`
	Label      string                                                                `json:"label"`
	Schema     map[string]any                                                        `json:"schema"`
	FieldOrder []string                                                              `json:"fieldOrder"`
	UI         map[string]any                                                        `json:"ui"`
}

type Catalog interface{ Definitions() []Definition }

type Preparer interface {
	Prepare(ctx context.Context, code string, config map[string]any) (map[string]any, string, error)
}

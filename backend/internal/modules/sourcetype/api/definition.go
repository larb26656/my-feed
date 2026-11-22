package api

// Definition describes accepted input and its presentation to readers.
type Definition struct {
	Normalize func(map[string]any) (map[string]any, string, error) `json:"-"`
	Obtainer  Obtainer                                             `json:"-"`
	Code      string                                               `json:"code"`
	Label     string                                               `json:"label"`
	Schema    map[string]any                                       `json:"schema"`
	UI        map[string]any                                       `json:"ui"`
}

type Catalog interface{ Definitions() []Definition }

type Preparer interface {
	Prepare(code string, config map[string]any) (map[string]any, string, error)
}

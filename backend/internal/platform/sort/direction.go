package sort

import "fmt"

// Direction is the order in which values are sorted.
type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

func (d Direction) Valid() bool { return d == Asc || d == Desc }

// UnmarshalParam binds an optional sort direction from a query parameter.
func (d *Direction) UnmarshalParam(value string) error {
	direction := Direction(value)
	if direction != "" && !direction.Valid() {
		return fmt.Errorf("invalid sort direction: %q", value)
	}
	*d = direction
	return nil
}

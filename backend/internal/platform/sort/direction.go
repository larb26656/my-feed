package sort

// Direction is the order in which values are sorted.
type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

func (d Direction) Valid() bool { return d == Asc || d == Desc }

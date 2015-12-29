package ds4util

// Filter filters its input.
type Filter interface {
	Filter([]int) []int
}

// Input is a filter that returns its input as output.
var Input Filter = &input{}

type input struct{}

func (*input) Filter(v []int) []int {
	return v
}

type combined struct {
	v []Filter
}

func Combine(f ...Filter) Filter {
	return &combined{f}
}

func (f *combined) Filter(v []int) []int {
	for _, x := range f.v {
		v = x.Filter(v)
	}
	return v
}

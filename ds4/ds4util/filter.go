package ds4util

type Filter interface {
	Filter(int) int
}

type combined struct {
	v []Filter
}

func Combine(f ...Filter) Filter {
	return &combined{f}
}

func (f *combined) Filter(v int) int {
	for _, x := range f.v {
		v = x.Filter(v)
	}
	return v
}

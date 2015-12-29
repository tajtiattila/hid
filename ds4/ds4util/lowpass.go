package ds4util

type AlphaFilter struct {
	Alpha float64
	v     int
}

func NewAlphaFilter(a float64) *AlphaFilter {
	return &AlphaFilter{Alpha: a}
}

func (f *AlphaFilter) Filter(v int) int {
	f.v += int(alpha * float64(v-f.v))
	return f.v
}

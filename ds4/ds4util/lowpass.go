package ds4util

type AlphaFilter struct {
	A float64
	V []int
}

// NewAlphaFilter returns an AlphaFilter
// for n values using a for the alpha.
func NewAlphaFilter(n int, a float64) *AlphaFilter {
	return &AlphaFilter{A: a, V: make([]int, n)}
}

func (f *AlphaFilter) Filter(v []int) []int {
	for i := range v {
		f.V[i] += int(f.A * float64(v[i]-f.V[i]))
	}
	return f.V
}

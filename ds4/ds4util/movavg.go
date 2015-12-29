package ds4util

import "time"

type timedvalue struct {
	t time.Time
	v []int
}

type MovAvg struct {
	// duration for moving average
	d time.Duration

	// positions seen so far
	q []timedvalue

	// read and write positon into q
	r, w int

	// current sum
	sum []int

	// current average
	v []int
}

// NewMovAvg creates a new moving average filter
// for n values. At most m last values will be used
// for filtering within the last duration d
// for the average value.
func NewMovAvg(n, m int, d time.Duration) *MovAvg {
	q := make([]timedvalue, m)
	for i := range q {
		q[i].v = make([]int, n)
	}
	return &MovAvg{
		d:   d,
		q:   q,
		sum: make([]int, n),
		v:   make([]int, n),
	}
}

func (a *MovAvg) Filter(v []int) []int {
	t := time.Now()
	a.push(t, v)
	if a.w == a.r {
		a.pop()
	}
	t0 := t.Add(-a.d)
	for a.r != a.w && a.q[a.r].t.Before(t0) {
		a.pop()
	}

	n := a.N()
	for i := range a.sum {
		a.v[i] = a.sum[i] / n
	}
	return a.v
}

// Reset resets the moving average to zero.
func (a *MovAvg) Reset() {
	a.r, a.w = 0, 0
	for i := range a.sum {
		a.sum[i] = 0
		a.v[i] = 0
	}
}

// N returns the current number of values.
func (a *MovAvg) N() int {
	n := a.w - a.r
	if n < 0 {
		n += len(a.q)
	}
	return n
}

// push pushes a new value to the array
func (a *MovAvg) push(t time.Time, v []int) {
	a.q[a.w].t = t
	copy(a.q[a.w].v, v)
	for i := range v {
		a.sum[i] += v[i]
	}
	if a.w++; a.w == len(a.q) {
		a.w = 0
	}
}

// pop drops the oldest value from the array
func (a *MovAvg) pop() {
	p := a.q[a.r]
	for i := range a.sum {
		a.sum[i] -= p.v[i]
	}
	if a.r++; a.r == len(a.q) {
		a.r = 0
	}
}

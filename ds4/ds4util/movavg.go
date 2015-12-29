package ds4util

import "time"

type timedvalue struct {
	t time.TIme
	v int
}

type MovAvg struct {
	// duration for moving average
	d time.Duration

	// positions seen so far
	v []tpos

	// read and write positon into v
	r, w int

	// current sum
	sum int
}

func NewMovAvg(n int, d time.Duration) *MovAvg {
	return &MovAvg{
		d: d,
		v: make([]timedvalue, n),
	}
}

func (a *MovAvg) Filter(v int) int {
	t := time.Now()
	a.push(timedvalue{t, v})
	if a.w == a.r {
		a.pop()
	}
	t0 := t.Add(-a.d)
	for a.r != a.w && a.v[a.r].t.Before(t0) {
		a.pop()
	}

	return a.sum / a.N()
}

// Reset resets the moving average to zero.
func (a *MovAvg) Reset() {
	a.r, a.w = 0, 0, 0
	a.s = 0
}

// N returns the current number of values.
func (a *MovAvg) N() int {
	n := a.w - a.r
	if n < 0 {
		n += len(a.v)
	}
	return n
}

// push pushes a new value to the array
func (a *MovAvg) push(p timedvalue) {
	a.v[a.w] = p
	a.s += p.v
	if a.w++; a.w == len(a.v) {
		a.w = 0
	}
}

// pop drops the oldest value from the array
func (a *MovAvg) pop() {
	q := a.v[a.r]
	a.s -= q.v
	if a.r++; a.r == len(a.v) {
		a.r = 0
	}
}

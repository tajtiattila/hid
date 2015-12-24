package ds4util

import "time"

type tpos struct {
	t    time.Time
	x, y int
}

type posavg struct {
	d       time.Duration
	v       []tpos
	r, w, n int
	sx, sy  int64
}

func newPosAvg(n int, d time.Duration) *posavg {
	return &posavg{
		d: d,
		v: make([]tpos, n),
	}
}

func (a *posavg) Len() int {
	return a.n
}

func (a *posavg) Dist2(x, y int) int64 {
	if a.n == 0 {
		return 0
	}
	ax, ay := a.Avg()
	dx, dy := int64(x-ax), int64(y-ay)
	return dx*dx + dy*dy
}

func (a *posavg) Avg() (x, y int) {
	n64 := int64(a.n)
	return int(a.sx / n64), int(a.sy / n64)
}

func (a *posavg) Reset() {
	a.n, a.r, a.w = 0, 0, 0
	a.sx, a.sy = 0, 0
}

func (a *posavg) Push(p tpos) {
	a.push(p)
	if a.w == a.r {
		a.pop()
	}
	t := p.t.Add(-a.d)
	for a.r != a.w && a.v[a.r].t.Before(t) {
		a.pop()
	}
}

func (a *posavg) push(p tpos) {
	a.v[a.w] = p
	a.sx += int64(p.x)
	a.sy += int64(p.y)
	a.n++
	if a.w++; a.w == len(a.v) {
		a.w = 0
	}
}

func (a *posavg) pop() {
	q := a.v[a.r]
	a.sx -= int64(q.x)
	a.sy -= int64(q.y)
	a.n--
	if a.r++; a.r == len(a.v) {
		a.r = 0
	}
}

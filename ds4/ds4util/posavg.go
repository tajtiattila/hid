package ds4util

import "time"

type tpos struct {
	t    time.Time
	x, y int32
}

type PosAvg struct {
	d       time.Duration
	v       []tpos
	r, w, n int
	sx, sy  int64
}

func NewPosAvg(n int, d time.Duration) *PosAvg {
	return &PosAvg{
		d: d,
		v: make([]tpos, n),
	}
}

func (a *PosAvg) N() int { return a.n }

func (a *PosAvg) Value() (x, y int32) {
	nf := int64(a.n)
	return int32(a.sx / nf), int32(a.sy / nf)
}

func (a *PosAvg) Reset() {
	a.n, a.r, a.w = 0, 0, 0
	a.sx, a.sy = 0, 0
}

func (a *PosAvg) Push(x, y int32) {
	t := time.Now()
	a.push(tpos{t, x, y})
	if a.w == a.r {
		a.pop()
	}
	t0 := t.Add(-a.d)
	for a.r != a.w && a.v[a.r].t.Before(t0) {
		a.pop()
	}
}

func (a *PosAvg) push(p tpos) {
	a.v[a.w] = p
	a.sx += int64(p.x)
	a.sy += int64(p.y)
	a.n++
	if a.w++; a.w == len(a.v) {
		a.w = 0
	}
}

func (a *PosAvg) pop() {
	q := a.v[a.r]
	a.sx -= int64(q.x)
	a.sy -= int64(q.y)
	a.n--
	if a.r++; a.r == len(a.v) {
		a.r = 0
	}
}

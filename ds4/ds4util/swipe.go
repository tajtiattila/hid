package ds4util

import (
	"time"

	"github.com/tajtiattila/hid/ds4"
)

const (
	SwipeUp = 1 << iota
	SwipeDown
	SwipeLeft
	SwipeRight

	swipeMask = SwipeUp | SwipeDown | SwipeLeft | SwipeRight
)

type SwipeHandler interface {
	Swipe(dir int, ntouch int)
	Touch(x, y int)
	Click(x, y int)
}

// SwipeLogic provides swipe, touch and click input.
type SwipeLogic struct {
	Handler SwipeHandler

	state swipeFunc

	// swipeStart data
	t0     time.Time // current touch start time
	x0, y0 int
	id     int // touch id

	// swipeBegin data
	xsum, ysum int64 // current sum of positions
	n          int64 // positions summed so far

	// swipeSwipe data
	ntouch int
}

func NewSwipeLogic(h SwipeHandler) *SwipeLogic {
	return &SwipeLogic{
		Handler: h,
		state:   swipeStart,
	}
}

func (l *SwipeLogic) HandleState(s *ds4.State) {
	l.state = l.state(l, s)
}

type swipeFunc func(l *SwipeLogic, s *ds4.State) swipeFunc

const (
	// a swipe is started when the first finger moved at
	// least swipeStartDist within swipeBeginTime
	swipeStartDist = 50
	swipeBeginTime = 50 * time.Microsecond

	// swipeDist is the travel distance needed for a swipe
	swipeDist = 300

	// swipeMaxTime is the longest time to travel swipeDist
	swipeMaxTime = 300 * time.Microsecond
)

// swipeStart is the start state when no touch or click is active
func swipeStart(l *SwipeLogic, s *ds4.State) swipeFunc {
	if s.Touch[0].Active() {
		x, y := int(s.Touch[0].X), int(s.Touch[0].Y)
		if s.Button&ds4.Click != 0 {
			// pad clicked
			l.Handler.Click(x, y)
			return swipeClear
		}
		l.t0, l.x0, l.y0, l.id = time.Now(), x, y, int(s.Touch[0].Id)
		l.xsum, l.ysum, l.n = int64(x), int64(y), 1
		return swipeBegin
	}
	return swipeStart
}

func swipeBegin(l *SwipeLogic, s *ds4.State) swipeFunc {
	x, y := int(s.Touch[0].X), int(s.Touch[0].Y)
	if s.Button&ds4.Click != 0 {
		// pad clicked
		l.Handler.Click(x, y)
		return swipeClear
	}
	if !s.Touch[0].Active() {
		// released
		l.Handler.Touch(x, y)
		return swipeClear
	}
	if int(s.Touch[0].Id) != l.id {
		return swipeClear
	}
	//ax, ay := int(l.xsum/l.n), int(l.ysum/l.n)
	dx, dy := int64(x-l.x0), int64(y-l.y0)
	if dx*dx+dy*dy > swipeStartDist*swipeStartDist {
		// swipe started
		l.ntouch = 1
		return swipeSwipe
	}
	t := time.Now()
	if t.Sub(l.t0) > swipeBeginTime {
		// touching still near start pos
		l.Handler.Touch(x, y)
		return swipeTouch
	}
	// still undecided
	return swipeBegin
}

func swipeSwipe(l *SwipeLogic, s *ds4.State) swipeFunc {
	if s.Button&ds4.Click != 0 {
		// click during swipe
		return swipeClear
	}
	if s.Touch[1].Active() {
		l.ntouch = 2
	}
	if s.Touch[0].Active() && int(s.Touch[0].Id) != l.id {
		// start touch replaced
		return swipeClear
	}

	x, y := int(s.Touch[0].X), int(s.Touch[0].Y)
	dx, dy := int64(x-l.x0), int64(y-l.y0)
	if dx*dx+dy*dy > swipeDist*swipeDist {
		if iabs(int(dx)) > iabs(int(dy)) {
			if dx > 0 {
				l.Handler.Swipe(SwipeRight, l.ntouch)
			} else {
				l.Handler.Swipe(SwipeLeft, l.ntouch)
			}
		} else {
			if dy > 0 {
				l.Handler.Swipe(SwipeDown, l.ntouch)
			} else {
				l.Handler.Swipe(SwipeUp, l.ntouch)
			}
		}
		return swipeClear
	}

	if time.Now().Sub(l.t0) > swipeMaxTime {
		return swipeClear
	}

	return swipeSwipe
}

// swipeTouch handles continuous touch that is not a swipe
func swipeTouch(l *SwipeLogic, s *ds4.State) swipeFunc {
	if int(s.Touch[0].Id) != l.id {
		return swipeClear
	}
	x, y := int(s.Touch[0].X), int(s.Touch[0].Y)
	if s.Button&ds4.Click != 0 {
		l.Handler.Click(x, y)
		return swipeClear
	}
	l.Handler.Touch(x, y)
	return swipeTouch
}

// swipeClear waits until no touch or click is active
func swipeClear(l *SwipeLogic, s *ds4.State) swipeFunc {
	if s.Button&ds4.Click != 0 {
		return swipeClear
	}
	if s.Touch[0].Active() || s.Touch[1].Active() {
		return swipeClear
	}
	return swipeStart
}

func iabs(x int) (r int) {
	if x >= 0 {
		r = x
	} else {
		r = -x
	}
	return
}

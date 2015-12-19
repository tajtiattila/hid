package ds4

import (
	"fmt"
	"time"
)

const (
	SwipeUp = 1 << iota
	SwipeDown
	SwipeLeft
	SwipeRight

	swipeMask = SwipeUp | SwipeDown | SwipeLeft | SwipeRight
)

type TouchHandler interface {
	Swipe(nfingers int, direction int)
	At(x, y int)
	Click(x, y int)
	Scroll(dx, dy int)
}

type Touchpad struct {
	l    swipeLogic
	tpkt byte
}

func NewTouchpad() *Touchpad {
	return &Touchpad{
		l: swipeLogic{
			state: swipeStart,
		},
	}
}

func (t *Touchpad) Tick(s *State) {
	if s.Packet != t.tpkt {
		t.tpkt = s.Packet
		t.l.tick(s)
	}
}

type Touchpadx struct {
	lastntouch int
	tstart     time.Time
	nmaxtouch  int
	maybeswipe byte

	start, last Touch
}

func (t *Touchpadx) Handle(s *State) {
	ntouch := s.NTouch()
	if t.lastntouch == 0 && ntouch == 0 {
		return
	}
	lastntouch := t.lastntouch
	t.lastntouch = ntouch
	if lastntouch == 0 {
		t.tstart = time.Now()
		t.nmaxtouch = ntouch
		t.maybeswipe = swipeMask
		t.start = s.Touch[0]
		return
	} else if ntouch == 0 {
		t.checkSwipe()
		return
	}

	if t.maybeswipe != 0 {
		startf := s.Finger(t.start.Id)
		if startf != nil {
			dx := t.last.X - t.start.X
			dy := t.last.Y - t.start.Y
			switch {
			case dx < 0:
				t.maybeswipe &^= SwipeRight
			case dx > 0:
				t.maybeswipe &^= SwipeLeft
			}
			switch {
			case dy < 0:
				t.maybeswipe &^= SwipeDown
			case dy > 0:
				t.maybeswipe &^= SwipeUp
			}
			t.last = *startf
		} else {
			// start finger lost
		}
	}

	if t.nmaxtouch < ntouch {
		t.nmaxtouch = ntouch
	}
}

func (t *Touchpadx) checkSwipe() {
	if t.maybeswipe != 0 {
		//dx := t.last.X - t.start.X
		//dy := t.last.Y - t.start.Y
	}
}

var maxSwipeDuration = 350 * time.Millisecond

type swipeLogic struct {
	state     swipeFunc
	ntouch    int
	start     time.Time
	swipe     byte
	dx, dy    int
	touch     [2]Touch
	nmaxtouch int
}

func (l *swipeLogic) tick(s *State) {
	l.state = l.state(l, s)
}

func (l *swipeLogic) check(cur, old Touch) bool {
	if cur.Id != old.Id {
		return false
	}
	dx := cur.X - old.X
	dy := cur.Y - old.Y
	switch {
	//case dx == 0:
	//l.swipe &^= SwipeLeft | SwipeRight
	case dx < 0:
		l.swipe &^= SwipeRight
	case dx > 0:
		l.swipe &^= SwipeLeft
	}
	switch {
	//case dy == 0:
	//l.swipe &^= SwipeUp | SwipeDown
	case dy < 0:
		l.swipe &^= SwipeDown
	case dy > 0:
		l.swipe &^= SwipeUp
	}
	l.dx += int(dx)
	l.dy += int(dy)
	return l.swipe != 0
}

func (l *swipeLogic) finish() swipeFunc {
	debug("finish")
	ax, ay := iabs(l.dx), iabs(l.dy)
	if ax > ay {
		if ax > 300 {
			switch {
			case l.dx < 0 && l.swipe&SwipeLeft != 0:
				l.handleSwipe(SwipeLeft)
			case 0 < l.dx && l.swipe&SwipeRight != 0:
				l.handleSwipe(SwipeRight)
			}
			return swipeStart
		}
	} else {
		if ay > 300 {
			switch {
			case l.dy < 0 && l.swipe&SwipeUp != 0:
				l.handleSwipe(SwipeUp)
			case 0 < l.dy && l.swipe&SwipeDown != 0:
				l.handleSwipe(SwipeDown)
			}
			return swipeStart
		}
	}

	l.handleTouch(l.touch[0])
	return swipeStart
}

func (l *swipeLogic) handleTouch(t Touch) {
	fmt.Println("Touch:", t.X, t.Y)
}

func (l *swipeLogic) handleSwipe(dir int) {
	fmt.Println(l.dx, l.dy, "Swipe:", dir, "Touches:", l.nmaxtouch)
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

type swipeFunc func(l *swipeLogic, c *State) swipeFunc

func swipeStart(l *swipeLogic, s *State) swipeFunc {
	ntouch := s.NTouch()
	if ntouch != 0 {
		if ntouch > 1 {
			l.touch[1] = s.Touch[1]
		}
		l.touch[0] = s.Touch[0]
		l.ntouch = ntouch
		l.nmaxtouch = ntouch
		l.start = time.Now()
		l.dx, l.dy = 0, 0
		l.swipe = swipeMask
		debug("starting")
		return swipeProgress
	}
	return swipeStart
}

//var debug = fmt.Println
func debug(a ...interface{}) {}

func swipeProgress(l *swipeLogic, s *State) swipeFunc {
	if time.Now().Sub(l.start) > maxSwipeDuration {
		// too long
		return swipeExpire
	}
	ntouch := s.NTouch()

	// verify old touches and take new ones
	switch ntouch {
	case 0:
		return l.finish()
	case 1:
		if l.ntouch == 1 {
			if s.Touch[0].Id != l.touch[0].Id {
				debug("finger lost")
				return swipeStart
			}
		} else { // l.ntouch == 2
			if s.Touch[0].Id == l.touch[1].Id {
				l.touch[0] = l.touch[1]
				l.ntouch = 1
			} else if s.Touch[0].Id != l.touch[0].Id {
				// aborted, lost finger
				debug("finger lost")
				return swipeStart
			}
			if !l.check(s.Touch[0], l.touch[0]) {
				debug("finger lost")
				return swipeStart
			}
			return swipeFinish
		}
	case 2:
		if l.ntouch == 1 {
			// add new touch
			switch {
			case l.touch[0].Id == s.Touch[0].Id:
				l.touch[1] = s.Touch[1]
			case l.touch[0].Id == s.Touch[1].Id:
				l.touch[1] = l.touch[0]
				l.touch[0] = s.Touch[0]
			default:
				debug("finger lost")
				return swipeStart
			}
			l.ntouch = 2
			l.nmaxtouch = 2
		} else { // l.ntouch == 2
			if l.touch[0].Id == s.Touch[1].Id &&
				l.touch[1].Id == s.Touch[0].Id {
				// swap
				l.touch[0], l.touch[1] = l.touch[1], l.touch[0]
			} else if !(l.touch[0].Id == s.Touch[0].Id &&
				l.touch[1].Id == s.Touch[1].Id) {
				// at lease one finger changed
				debug("finger lost")
				return swipeStart
			}
		}
	}

	if !l.check(s.Touch[0], l.touch[0]) {
		debug("swipe canceled")
		return swipeExpire
	}

	if ntouch == 2 {
		l.touch[1] = s.Touch[1]
	}
	l.touch[0] = s.Touch[0]

	return swipeProgress
}

func swipeFinish(l *swipeLogic, s *State) swipeFunc {
	if time.Now().Sub(l.start) > maxSwipeDuration {
		// too long
		return swipeExpire
	}

	ntouch := s.NTouch()
	switch ntouch {
	case 0:
		return l.finish()
	case 2:
		return swipeExpire
	}
	if !l.check(s.Touch[0], l.touch[0]) {
		return swipeExpire
	}
	l.touch[0] = s.Touch[0]

	return swipeFinish
}

func swipeExpire(l *swipeLogic, s *State) swipeFunc {
	if s.NTouch() == 0 {
		return swipeStart
	}
	if s.Touch[0] != l.touch[0] {
		l.handleTouch(s.Touch[0])
		l.touch[0] = s.Touch[0]
	}
	return swipeExpire
}

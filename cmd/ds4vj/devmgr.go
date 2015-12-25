package main

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/tajtiattila/hid"
	"github.com/tajtiattila/hid/ds4"
)

const (
	ConnUSB = 1
	ConnBT  = 2
)

type DS4Entry struct {
	Name    string
	Serial  string
	Conn    int
	Battery string
}

func (s DS4Entry) String() string {
	var conn string
	if s.Conn == ConnUSB {
		conn = "USB"
	} else {
		conn = "BT"
	}
	return s.Serial + "(" + conn + ")"
}

type DS4Event struct {
	DS4Entry
	Removed bool
}

type StateHandler interface {
	// State is called each time new input arrives
	State(s *ds4.State) error

	// Close is called after a device is not used anymore.
	Close() error
}

// ConnectHandler handles new DS4 devices. It should perform initialization
// on d and return a StateHandler.
type ConnectHandler func(d *ds4.Device, e DS4Entry, log *log.Logger) (StateHandler, error)

type DeviceManager struct {
	// protects dev
	mtx sync.RWMutex
	dev map[string]DS4Entry

	log *log.Logger
	che chan DS4Event
	chq chan struct{}

	connh ConnectHandler
}

func NewDeviceManager(h ConnectHandler) *DeviceManager {
	m := &DeviceManager{
		dev:   make(map[string]DS4Entry),
		che:   make(chan DS4Event),
		chq:   make(chan struct{}),
		connh: h,
	}
	go func() {
		for {
			m.findDevices()
			time.Sleep(time.Second)
		}
	}()
	return m
}

func (m *DeviceManager) Event() <-chan DS4Event {
	return m.che
}

func (m *DeviceManager) Close() error {
	close(m.che)
	close(m.chq)
	return nil
}

func (m *DeviceManager) findDevices() {
	dlist, err := hid.VendorDevices(0x54C, 0x5C4) // DualShock 4
	if err != nil {
		m.log.Println(err)
		return
	}

	sort.Sort(InputLenSort(dlist))

	for _, di := range dlist {
		if di.Attr.SerialNo == "" {
			continue
		}
		m.mtx.RLock()
		_, ok := m.dev[di.Attr.SerialNo]
		m.mtx.RUnlock()
		if !ok {
			m.runDevice(di)
		}
	}
}

func (m *DeviceManager) runDevice(di *hid.DeviceInfo) {
	d, err := ds4.Open(di.Name)
	if err != nil {
		m.log.Println(di.Attr.SerialNo, "opening device:", err)
		return
	}

	d.SetTimeout(time.Second)

	// read a few states before commencing
	var s ds4.State
	for i := 0; i < 10; i++ {
		if err := d.ReadState(&s); err != nil {
			m.log.Println(di.Attr.SerialNo, "initialization:", err)
			d.Close()
			return
		}
	}

	battery := s.Battery
	var conn int
	if d.Bluetooth() {
		conn = ConnBT
	} else {
		conn = ConnUSB
	}

	e := DS4Entry{
		Name:    di.Name,
		Serial:  di.Attr.SerialNo,
		Conn:    conn,
		Battery: batteryString(battery),
	}

	h, err := m.connh(d, e, m.log)
	if err != nil {
		m.log.Println(e, "handler init:", err)
		d.Close()
		return
	}

	m.mtx.Lock()
	if _, ok := m.dev[e.Serial]; ok {
		m.mtx.Unlock()
		d.Close()
		h.Close()
		return
	}
	m.dev[e.Serial] = e
	m.mtx.Unlock()

	m.che <- DS4Event{e, false}

	chq := m.chq

	go func() {
		defer func() {
			h.Close()
			d.Close()
		}()
		var s ds4.State
		for {
			select {
			case <-chq:
				d.DisconnectRadio()
				break
			}
			err := d.ReadState(&s)
			if err == nil {
				err = h.State(&s)
			}
			if err != nil {
				m.log.Println(e, "stopping:", err)
				break
			}
			if battery != s.Battery {
				// report new battery state
				battery = s.Battery
				e.Battery = batteryString(battery)
				m.che <- DS4Event{e, false}
			}
		}

		m.mtx.Lock()
		delete(m.dev, e.Serial)
		m.mtx.Unlock()

		m.che <- DS4Event{e, true}
	}()
}

func batteryString(b byte) string {
	var buf [16]byte
	p := buf[:0]
	if b&0xF0 != 0 {
		// charging indicator
		p = append(p, '+')
	}
	n := b & 0xF
	switch n {
	case 10:
		p = append(p, '1', '0', '0')
	case 0:
		p = append(p, '0')
	default:
		p = append(p, n, '0')
	}
	return string(append(p, '%'))
}

type InputLenSort []*hid.DeviceInfo

func (s InputLenSort) Len() int           { return len(s) }
func (s InputLenSort) Less(i, j int) bool { return s[i].Caps.InputLen > s[j].Caps.InputLen }
func (s InputLenSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
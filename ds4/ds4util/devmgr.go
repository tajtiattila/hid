package ds4util

import (
	"fmt"
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

type Entry struct {
	Name    string
	Serial  string
	Conn    int
	Battery byte
}

func (e *Entry) String() string {
	var conn string
	if e.Conn == ConnUSB {
		conn = "USB"
	} else {
		conn = "BT"
	}
	return e.Serial + "(" + conn + ")"
}

func (e *Entry) ConnString() string {
	switch e.Conn {
	case ConnUSB:
		return "USB"
	case ConnBT:
		return "BT"
	}
	return "?"
}

func (e *Entry) BatteryString() string {
	return batteryString(e.Battery)
}

// Charging reports if the battery is charging.
func (e *Entry) Charging() bool {
	return e.Battery&0xF0 != 0
}

// BatteryLevel reports the battery level percentage divided by 10.
func (e *Entry) BatteryLevel() byte {
	return e.Battery & 0x0F
}

// Event is sent when a new device is connected,
// an old device is disconnected, or its battery state changes.
type Event struct {
	Entry
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
type ConnectHandler interface {
	Connect(d *ds4.Device, e Entry) (StateHandler, error)
}

type DeviceManager struct {
	// protects dev
	mtx sync.RWMutex
	dev map[string]Entry

	log *log.Logger
	che chan Event

	chq chan chan struct{}

	chqwork chan struct{}
	grpwork sync.WaitGroup

	connh ConnectHandler
}

func NewDeviceManager(h ConnectHandler, log *log.Logger) *DeviceManager {
	m := &DeviceManager{
		dev:     make(map[string]Entry),
		che:     make(chan Event),
		chqwork: make(chan struct{}),
		chq:     make(chan chan struct{}),
		connh:   h,
		log:     log,
	}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				m.findDevices()
			case c := <-m.chq:
				close(m.chqwork)
				m.grpwork.Wait()
				close(m.che)
				close(c)
				return
			}
		}
	}()
	return m
}

func (m *DeviceManager) Event() <-chan Event {
	return m.che
}

func (m *DeviceManager) Close() error {
	c := make(chan struct{})
	m.chq <- c
	<-c
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
		m.log.Print("opening device ", di.Attr.SerialNo, ": ", err)
		return
	}

	d.SetTimeout(time.Second)

	// read a few states before commencing
	var s ds4.State
	for i := 0; i < 10; i++ {
		if err := d.ReadState(&s); err != nil {
			m.log.Print("initialization", di.Attr.SerialNo, ": ", err)
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

	e := Entry{
		Name:    di.Name,
		Serial:  di.Attr.SerialNo,
		Conn:    conn,
		Battery: battery,
	}

	h, err := m.connh.Connect(d, e)
	if err != nil {
		m.log.Print("handler init ", e.String(), ": ", err)
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

	m.che <- Event{e, false}
	m.grpwork.Add(1)

	m.log.Println("starting", e.String())

	chq := m.chqwork

	go func() {
		defer func() {
			h.Close()
			d.Close()
			m.log.Print("stopping ", e.String(), ": ", err)

			m.mtx.Lock()
			delete(m.dev, e.Serial)
			m.mtx.Unlock()

			m.che <- Event{e, true}
			m.grpwork.Done()
		}()
		var s ds4.State
		for {
			select {
			case <-chq:
				d.DisconnectRadio()
				break
			default:
			}
			err := d.ReadState(&s)
			if err == nil {
				err = h.State(&s)
			}
			if err != nil {
				break
			}
			if battery != s.Battery {
				// report new battery state
				battery = s.Battery
				e.Battery = battery
				m.che <- Event{e, false}
			}
		}
	}()
}

func batteryString(b byte) string {
	return batstr[int(b&0x1F)]
}

var batstr []string

func init() {
	batstr = make([]string, 32)
	for i := 0; i < 16; i++ {
		j := i
		if j > 10 {
			j = 10
		}
		s := fmt.Sprint(j*10, "%+")
		batstr[i] = s[:len(s)-1]
		batstr[i+16] = s
	}
}

type InputLenSort []*hid.DeviceInfo

func (s InputLenSort) Len() int           { return len(s) }
func (s InputLenSort) Less(i, j int) bool { return s[i].Caps.InputLen > s[j].Caps.InputLen }
func (s InputLenSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

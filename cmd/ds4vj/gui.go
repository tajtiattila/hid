package main

import (
	"io"
	"os"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var AppTitle = "ds4vj"

func guimain(f func(w io.Writer, ch chan<- DS4Event)) {
	app := walk.App()

	// These specify the app data sub directory for the settings file.
	app.SetOrganizationName("Attila Tajti")
	app.SetProductName(AppTitle)
	settings := walk.NewIniFileSettings("gui.ini")
	settings.SetExpireDuration(time.Hour * 24 * 30 * 3)

	if err := settings.Load(); err == nil {
		app.SetSettings(settings)
	}

	model := new(DS4TableModel)

	var mw *walk.MainWindow
	var lv *LogView
	if err := (MainWindow{
		Title:    AppTitle,
		Name:     "mainWindow",
		AssignTo: &mw,
		MenuItems: []MenuItem{
			Menu{
				Text: "File",
				Items: []MenuItem{
					Action{
						Text:        "Exit",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
		},
		MinSize: Size{600, 400},
		Layout:  VBox{MarginsZero: true},
		Children: []Widget{
			HSplitter{
				Name: "splitter",
				Children: []Widget{
					TableView{
						Name:             "table",
						ColumnsOrderable: true,
						Columns: []TableViewColumn{
							{Title: "Serial", Name: "serial"},
							{Title: "Conn", Name: "conn", Width: 100},
							{Title: "Battery", Name: "battery", Width: 100, Alignment: AlignFar},
						},
						Model: model,
					},
					declLogView{AssignTo: &lv},
				},
			},
		},
	}).Create(); err != nil {
		Fatal(err)
	}

	ch := make(chan DS4Event)
	go func() {
		for e := range ch {
			mw.Synchronize(func() {
				model.Handle(e)
			})
		}
		walk.MsgBox(nil, "Fatal", "Worker exited", walk.MsgBoxIconExclamation)
		mw.Close()
	}()

	f(lv, ch)

	defer settings.Save()
	mw.Run()
}

func Fatal(err error) {
	walk.MsgBox(nil, "Fatal", err.Error(), walk.MsgBoxIconExclamation)
	os.Exit(1)
}

type DS4TableModel struct {
	walk.TableModelBase

	items []*DS4Entry
}

func (m *DS4TableModel) RowCount() int {
	return len(m.items)
}

func (m *DS4TableModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.Serial

	case 1:
		switch item.Conn {
		case ConnUSB:
			return "USB"
		case ConnBT:
			return "BT"
		default:
			return "?"
		}

	case 2:
		return item.Battery
	}

	panic("unexpected col")
}

func (m *DS4TableModel) Handle(e DS4Event) {
	if e.Removed {
		for i, item := range m.items {
			if item.Serial == e.Serial && item.Conn == e.Conn {
				copy(m.items[i:], m.items[i+1:])
				m.PublishRowsReset()
				return
			}
		}
	} else {
		for i, item := range m.items {
			if item.Serial == e.Serial && item.Conn == e.Conn {
				if item.Battery != e.Battery {
					item.Battery = e.Battery
					m.PublishRowChanged(i)
				}
				return
			}
		}
		// not found
		p := new(DS4Entry)
		*p = e.DS4Entry
		m.items = append(m.items, p)
		m.PublishRowsReset()
	}
}

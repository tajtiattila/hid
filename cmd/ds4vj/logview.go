// Copyright 2012 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"bytes"
	"errors"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type declLogView struct {
	AssignTo **LogView
	declarative.CustomWidget
}

func (lv declLogView) Create(b *declarative.Builder) error {
	w, err := NewLogView(b.Parent())
	if err != nil {
		return err
	}

	lv.MinSize = declarative.Size{300, 200}
	return b.InitWidget(lv, w, func() error {
		if lv.AssignTo != nil {
			*lv.AssignTo = w
		}
		return nil
	})
}

type LogView struct {
	walk.WidgetBase
	logChan chan string

	bufmtx sync.Mutex
	buf    CrLfBuf
}

const (
	TEM_APPENDTEXT = win.WM_USER + 6
)

func NewLogView(parent walk.Container) (*LogView, error) {
	lc := make(chan string, 1024)
	lv := &LogView{logChan: lc}

	if err := walk.InitWidget(
		lv,
		parent,
		"EDIT",
		win.WS_TABSTOP|win.WS_VISIBLE|win.WS_VSCROLL|win.ES_MULTILINE|win.ES_WANTRETURN,
		win.WS_EX_CLIENTEDGE); err != nil {
		return nil, err
	}
	lv.setReadOnly(true)
	lv.SendMessage(win.EM_SETLIMITTEXT, 1<<32-1, 0)
	return lv, nil
}

func (*LogView) LayoutFlags() walk.LayoutFlags {
	return walk.ShrinkableHorz | walk.ShrinkableVert | walk.GrowableHorz | walk.GrowableVert | walk.GreedyHorz | walk.GreedyVert
}

func (*LogView) MinSizeHint() walk.Size {
	return walk.Size{20, 12}
}

func (*LogView) SizeHint() walk.Size {
	return walk.Size{100, 100}
}

func (lv *LogView) Clear() {
	eol := uint16(0)
	lv.SendMessage(win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(&eol)))
}

func (lv *LogView) AppendText(value string) {
	// save old selection
	var selstart, selend int32
	lv.SendMessage(win.EM_GETSEL, uintptr(unsafe.Pointer(&selstart)), uintptr(unsafe.Pointer(&selend)))

	textLength := lv.SendMessage(win.WM_GETTEXTLENGTH, uintptr(0), uintptr(0))
	lv.SendMessage(win.EM_SETSEL, textLength, textLength)
	lv.SendMessage(win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(value))))

	// restore selection
	lv.SendMessage(win.EM_SETSEL, uintptr(selstart), uintptr(selend))
}

func (lv *LogView) setReadOnly(readOnly bool) error {
	if 0 == lv.SendMessage(win.EM_SETREADONLY, uintptr(win.BoolToBOOL(readOnly)), 0) {
		return errors.New("fail to call EM_SETREADONLY")
	}

	return nil
}

func (lv *LogView) PostAppendText(value string) {
	lv.logChan <- value
	win.PostMessage(lv.Handle(), TEM_APPENDTEXT, 0, 0)
}

func (lv *LogView) Write(p []byte) (int, error) {
	lv.bufmtx.Lock()
	defer lv.bufmtx.Unlock()
	lv.buf.Reset()
	lv.buf.Write(p)
	lv.PostAppendText(lv.buf.String())
	return len(p), nil
}

func (lv *LogView) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_GETDLGCODE:
		if wParam == win.VK_RETURN {
			return win.DLGC_WANTALLKEYS
		}

		return win.DLGC_HASSETSEL | win.DLGC_WANTARROWS | win.DLGC_WANTCHARS
	case TEM_APPENDTEXT:
		select {
		case value := <-lv.logChan:
			lv.AppendText(value)
		default:
			return 0
		}
	}

	return lv.WidgetBase.WndProc(hwnd, msg, wParam, lParam)
}

type CrLfBuf struct {
	bytes.Buffer
	lastc uint8
}

func (b *CrLfBuf) Write(p []byte) (int, error) {
	for _, c := range p {
		if c == '\n' && b.lastc != '\r' {
			b.WriteByte('\r')
		}
		b.WriteByte(c)
		b.lastc = c
	}
	return len(p), nil
}

package x11

import (
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgb/xtest"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
)

type Inputter struct {
	xu *xgbutil.XUtil
}

func NewInputter(xu *xgbutil.XUtil) (*Inputter, error) {
	if err := xtest.Init(xu.Conn()); err != nil {
		return nil, err
	}
	keybind.Initialize(xu)
	return &Inputter{
		xu: xu,
	}, nil
}

func (i *Inputter) Move(x, y int) {
	xproto.WarpPointer(i.xu.Conn(), 0, i.xu.RootWin(), 0, 0, 0, 0, int16(x), int16(y))
	i.xu.Sync()
}

func (i *Inputter) SendKey(root xproto.Window, keycode xproto.Keycode, isPress bool) {
	inputType := xproto.KeyRelease
	if isPress {
		inputType = xproto.KeyPress
	}
	xtest.FakeInput(i.xu.Conn(), byte(inputType), byte(keycode), xproto.TimeCurrentTime, root, 0, 0, 0)
	i.xu.Sync()
}

func (i *Inputter) SendButton(root xproto.Window, button xproto.Button, isPress bool) {
	inputType := xproto.ButtonRelease
	if isPress {
		inputType = xproto.ButtonPress
	}
	xtest.FakeInput(i.xu.Conn(), byte(inputType), byte(button), xproto.TimeCurrentTime, root, 0, 0, 0)
	i.xu.Sync()
}

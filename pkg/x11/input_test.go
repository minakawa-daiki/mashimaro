package x11

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/BurntSushi/xgbutil/keybind"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
)

func newDefaultInputter(t *testing.T) *Inputter {
	xu, err := xgbutil.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	i, err := NewInputter(xu)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestMove(t *testing.T) {
	i := newDefaultInputter(t)
	i.Move(0, 0)
}

func TestSendKey(t *testing.T) {
	i := newDefaultInputter(t)
	StartWineProcess(t, i.xu, "notepad")

	for _, char := range "hello" {
		for _, keycode := range keybind.StrToKeycodes(i.xu, string(char)) {
			i.SendKey(i.xu.RootWin(), keycode, true)
			i.SendKey(i.xu.RootWin(), keycode, false)
		}
	}
}

func TestButton(t *testing.T) {
	i := newDefaultInputter(t)
	cmd := StartWineProcess(t, i.xu, "notepad")
	windows, err := EnumWindowsByPid(i.xu, cmd.Process.Pid, i.xu.RootWin(), true)
	assert.NoError(t, err)
	mainWindow, err := GetMainWindow(i.xu, windows)
	assert.NoError(t, err)
	err = ActivateWindow(i.xu, mainWindow, true)
	assert.NoError(t, err)
	screenX, screenY, err := GetWindowPositionOnScreen(i.xu, mainWindow)
	assert.NoError(t, err)
	i.Move(screenX+10, screenY+10)
	time.Sleep(50 * time.Millisecond)
	i.SendButton(i.xu.RootWin(), xproto.ButtonIndex3, true)
	i.SendButton(i.xu.RootWin(), xproto.ButtonIndex3, false)
	time.Sleep(100 * time.Millisecond)
}

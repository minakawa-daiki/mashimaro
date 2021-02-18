package x11

import (
	"testing"

	"github.com/BurntSushi/xgbutil/ewmh"

	"github.com/stretchr/testify/assert"

	"github.com/BurntSushi/xgbutil"
)

func TestEnumWindows(t *testing.T) {
	xu, err := xgbutil.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	cmd := StartWineProcess(t, xu, "notepad", "unknown_file")
	windows, err := EnumWindowsByPid(xu, cmd.Process.Pid, xu.RootWin(), true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, windows, 2)
}

func TestGetMainWindow(t *testing.T) {
	xu, err := xgbutil.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	cmd := StartWineProcess(t, xu, "notepad", "unknown_file")
	windows, err := EnumWindowsByPid(xu, cmd.Process.Pid, xu.RootWin(), true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, windows, 2)

	mainWindow, err := GetMainWindow(xu, windows)
	assert.NoError(t, err)
	width, height, err := GetWindowSize(xu, mainWindow)
	assert.NoError(t, err)
	assert.True(t, width > 0)
	assert.True(t, height > 0)
	mainWindowTitle, err := ewmh.WmNameGet(xu, mainWindow)
	assert.NoError(t, err)
	assert.Equal(t, "Untitled - Notepad", mainWindowTitle)

	x, y, err := GetWindowPositionOnScreen(xu, xu.Screen(), mainWindow)
	assert.NoError(t, err)
	assert.True(t, x > 0)
	assert.True(t, y > 0)
}

func TestCenterWindow(t *testing.T) {
	xu, err := xgbutil.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	cmd := StartWineProcess(t, xu, "notepad", "unknown_file")
	windows, err := EnumWindowsByPid(xu, cmd.Process.Pid, xu.RootWin(), true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, windows, 2)

	for _, window := range windows {
		err := CenterWindow(xu, xu.Screen(), window, true)
		assert.NoError(t, err)
	}
}

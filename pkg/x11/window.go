package x11

import (
	"github.com/pkg/errors"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
)

func EnumWindows(xu *xgbutil.XUtil, root xproto.Window, onlyVisible bool) ([]xproto.Window, error) {
	var windows []xproto.Window
	reply, err := xproto.QueryTree(xu.Conn(), root).Reply()
	var windowErr xproto.WindowError
	if errors.As(err, &windowErr) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, w := range reply.Children {
		isVisible, err := IsWindowVisible(xu, w)
		if err != nil {
			return nil, err
		}
		if !onlyVisible || isVisible {
			windows = append(windows, w)
		}
		children, err := EnumWindows(xu, w, onlyVisible)
		if err != nil {
			return nil, err
		}
		windows = append(windows, children...)
	}
	return windows, nil
}

func EnumWindowsByPid(xu *xgbutil.XUtil, pid int, root xproto.Window, onlyVisible bool) ([]xproto.Window, error) {
	var windows []xproto.Window
	reply, err := xproto.QueryTree(xu.Conn(), root).Reply()
	var windowErr xproto.WindowError
	if errors.As(err, &windowErr) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, w := range reply.Children {
		// ignore error to skip when could not get pid
		wpid, _ := ewmh.WmPidGet(xu, w)
		if wpid > 0 && wpid == uint(pid) {
			isVisible, err := IsWindowVisible(xu, w)
			if err != nil {
				return nil, err
			}
			if !onlyVisible || isVisible {
				windows = append(windows, w)
			}
		}
		children, err := EnumWindowsByPid(xu, pid, w, onlyVisible)
		if err != nil {
			return nil, err
		}
		windows = append(windows, children...)
	}
	return windows, nil
}

func IsWindowVisible(xu *xgbutil.XUtil, window xproto.Window) (bool, error) {
	reply, err := xproto.GetWindowAttributes(xu.Conn(), window).Reply()
	if err != nil {
		return false, err
	}
	return reply.MapState == xproto.MapStateViewable, nil
}

func GetWindowPositionOnScreen(xu *xgbutil.XUtil, screen *xproto.ScreenInfo, window xproto.Window) (x, y int, err error) {
	if _, err := xproto.GetGeometry(xu.Conn(), xproto.Drawable(window)).Reply(); err != nil {
		return 0, 0, err
	}
	offset, err := xproto.TranslateCoordinates(xu.Conn(), window, screen.Root, 0, 0).Reply()
	if err != nil {
		return 0, 0, err
	}
	return int(offset.DstX), int(offset.DstY), nil
}

func GetWindowSize(xu *xgbutil.XUtil, window xproto.Window) (width, height int, err error) {
	reply, err := xproto.GetGeometry(xu.Conn(), xproto.Drawable(window)).Reply()
	if err != nil {
		return 0, 0, err
	}
	return int(reply.Width), int(reply.Height), nil
}

func GetMainWindow(xu *xgbutil.XUtil, windows []xproto.Window) (xproto.Window, error) {
	maxSize := 0
	var mainWindow xproto.Window
	for _, w := range windows {
		reply, err := xproto.GetGeometry(xu.Conn(), xproto.Drawable(w)).Reply()
		if err != nil {
			return 0, err
		}
		size := int(reply.Width * reply.Height)
		if size > maxSize {
			maxSize = size
			mainWindow = w
		}
	}
	if mainWindow == 0 {
		return 0, errors.New("could not find main window")
	}
	return mainWindow, nil
}

func CenterWindow(xu *xgbutil.XUtil, screen *xproto.ScreenInfo, window xproto.Window, sync bool) error {
	winWidth, winHeight, err := GetWindowSize(xu, window)
	if err != nil {
		return err
	}
	x := (int(screen.WidthInPixels) - winWidth) / 2
	y := (int(screen.HeightInPixels) - winHeight) / 2
	values := []uint32{uint32(x), uint32(y)}
	xproto.ConfigureWindow(xu.Conn(), window, xproto.ConfigWindowX|xproto.ConfigWindowY, values)
	if sync {
		for {
			cx, cy, err := GetWindowPositionOnScreen(xu, xu.Screen(), window)
			if err != nil {
				return err
			}
			if cx == x && cy == y {
				break
			}
		}
	}
	return nil
}

func ActivateWindow(xu *xgbutil.XUtil, window xproto.Window, sync bool) error {
	if err := ewmh.ActiveWindowSet(xu, window); err != nil {
		return err
	}
	if sync {
		for {
			current, err := ewmh.ActiveWindowGet(xu)
			if err != nil {
				return err
			}
			if current == window {
				break
			}
		}
	}
	return nil
}

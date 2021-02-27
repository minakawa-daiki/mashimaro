package gameserver

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"
)

var (
	errNoWindows = errors.New("no windows")
)

func (s *GameServer) startWatchGame(ctx context.Context, pub *captureRectPubSub) error {
	return s.startWatchCaptureRect(ctx, pub)
}

func (s *GameServer) startWatchCaptureRect(ctx context.Context, pub *captureRectPubSub) error {
	xu, err := xgbutil.NewConn()
	if err != nil {
		return err
	}
	var currentRect ScreenRect
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			rect, err := getMainWindowRect(xu)
			if err == errNoWindows {
				continue
			}
			if err != nil {
				log.Printf("failed to get main window rect: %+v", err)
				continue
			}
			if rect.IsValid() && rectHasChanged(rect, &currentRect) {
				currentRect = *rect
				pub.Publish(*rect)
			}
		}
	}
}

func getMainWindowRect(xu *xgbutil.XUtil) (*ScreenRect, error) {
	windows, err := x11.EnumWindows(xu, xu.RootWin(), true)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return nil, errNoWindows
	}
	mainWindow, err := x11.GetMainWindow(xu, windows)
	if err != nil {
		return nil, err
	}
	x, y, err := x11.GetWindowPositionOnScreen(xu, xu.Screen(), mainWindow)
	if err != nil {
		return nil, err
	}
	width, height, err := x11.GetWindowSize(xu, mainWindow)
	if err != nil {
		return nil, err
	}
	return &ScreenRect{
		StartX: x,
		StartY: y,
		EndX:   x + width,
		EndY:   y + height,
	}, nil
}

func rectHasChanged(a1, a2 *ScreenRect) bool {
	return a1.StartX != a2.StartX ||
		a1.StartY != a2.StartY ||
		a1.EndX != a2.EndX ||
		a1.EndY != a2.EndY
}

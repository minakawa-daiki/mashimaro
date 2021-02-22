package gameserver

import (
	"context"
	"log"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"
)

var (
	errNoWindows = errors.New("no windows")
)

func (s *GameServer) startWatchGame(ctx context.Context, pub *captureAreaPubSub) error {
	return s.startListenCaptureArea(ctx, pub)
}

func (s *GameServer) startListenCaptureArea(ctx context.Context, pub *captureAreaPubSub) error {
	xu, err := xgbutil.NewConn()
	if err != nil {
		return err
	}
	var captureArea streamer.ScreenCaptureArea
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			area, err := getMainWindowArea(xu)
			if err == errNoWindows {
				continue
			}
			if err != nil {
				log.Printf("failed to get main window area: %+v", err)
				continue
			}
			if area.IsValid() && areaHasChanged(area, &captureArea) {
				captureArea = *area
				pub.Publish(&captureArea)
			}
		}
	}
}

func getMainWindowArea(xu *xgbutil.XUtil) (*streamer.ScreenCaptureArea, error) {
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
	return &streamer.ScreenCaptureArea{
		StartX: x,
		StartY: y,
		EndX:   x + width - 1,
		EndY:   y + height - 1,
	}, nil
}

func areaHasChanged(a1, a2 *streamer.ScreenCaptureArea) bool {
	return a1.StartX != a2.StartX ||
		a1.StartY != a2.StartY ||
		a1.EndX != a2.EndX ||
		a1.EndY != a2.EndY
}

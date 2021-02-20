package gameserver

import (
	"context"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"

	"github.com/castaneai/mashimaro/pkg/streamer"
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
			windows, err := x11.EnumWindows(xu, xu.RootWin(), true)
			if err != nil {
				return err
			}
			if len(windows) == 0 {
				continue
			}
			mainWindow, err := x11.GetMainWindow(xu, windows)
			if err != nil {
				return err
			}
			x, y, err := x11.GetWindowPositionOnScreen(xu, xu.Screen(), mainWindow)
			if err != nil {
				return err
			}
			width, height, err := x11.GetWindowSize(xu, mainWindow)
			if err != nil {
				return err
			}
			area := &streamer.ScreenCaptureArea{
				StartX: x,
				StartY: y,
				EndX:   x + width - 1,
				EndY:   y + height - 1,
			}
			if area.IsValid() && areaHasChanged(area, &captureArea) {
				captureArea = *area
				pub.Publish(&captureArea)
			}
		}
	}
}

func areaHasChanged(a1, a2 *streamer.ScreenCaptureArea) bool {
	return a1.StartX != a2.StartX ||
		a1.StartY != a2.StartY ||
		a1.EndX != a2.EndX ||
		a1.EndY != a2.EndY
}

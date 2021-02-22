package gameserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/castaneai/mashimaro/pkg/proto"
)

var (
	errGameExited = errors.New("game exited")
)

func (s *GameServer) startController(ctx context.Context, message <-chan []byte, captureAreaChanged <-chan *streamer.ScreenCaptureArea) error {
	log.Printf("initialing x11 connection")
	xu, err := xgbutil.NewConn()
	if err != nil {
		return errors.Wrap(err, "failed to connect to X11")
	}
	xi, err := x11.NewInputter(xu)
	if err != nil {
		return errors.Wrap(err, "failed to new X11 inputter")
	}
	log.Printf("waiting for capture area")
	captureArea := <-captureAreaChanged
	log.Printf("start messaging")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-message:
			if err := s.handleMessage(ctx, msg, captureArea, xu, xi); err != nil {
				if err == errGameExited {
					return err
				}
				log.Printf("failed to handle message: %+v", err)
			}
		}
	}
}

func (s *GameServer) handleMessage(ctx context.Context, data []byte, captureArea *streamer.ScreenCaptureArea, xu *xgbutil.XUtil, xinput *x11.Inputter) error {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case MessageTypeMove:
		var body MoveMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		x := captureArea.StartX + body.X
		y := captureArea.StartY + body.Y
		xinput.Move(x, y)
		return nil
	case MessageTypeMouseDown:
		var body MouseDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendButton(xu.RootWin(), xproto.Button(body.Button), true)
		return nil
	case MessageTypeMouseUp:
		var body MouseUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendButton(xu.RootWin(), xproto.Button(body.Button), false)
		return nil
	case MessageTypeKeyDown:
		var body KeyDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendKey(xu.RootWin(), xproto.Keycode(body.Key), true)
		return nil
	case MessageTypeKeyUp:
		var body KeyUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendKey(xu.RootWin(), xproto.Keycode(body.Key), false)
		return nil
	case MessageTypeExitGame:
		if _, err := s.gameProcess.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
			return err
		}
		return errGameExited
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

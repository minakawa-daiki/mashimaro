package gameserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/castaneai/mashimaro/pkg/proto"
)

var (
	errGameExited = errors.New("game exited")
)

func (s *GameServer) startController(ctx context.Context, message <-chan []byte, captureRectChanged <-chan ScreenRect) error {
	log.Printf("initialing x11 connection")
	xu, err := xgbutil.NewConn()
	if err != nil {
		return errors.Wrap(err, "failed to connect to X11")
	}
	xi, err := x11.NewInputter(xu)
	if err != nil {
		return errors.Wrap(err, "failed to new X11 inputter")
	}
	log.Printf("start messaging")
	var captureRect *ScreenRect
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case rect := <-captureRectChanged:
			log.Printf("capture rect has changed: %s", &rect)
			captureRect = &rect
		case msg := <-message:
			if err := s.handleControllerMessage(ctx, msg, captureRect, xu, xi); err != nil {
				if err == errGameExited {
					return err
				}
				log.Printf("failed to handle message: %+v", err)
			}
		}
	}
}

func (s *GameServer) handleControllerMessage(ctx context.Context, data []byte, captureRect *ScreenRect, xu *xgbutil.XUtil, xinput *x11.Inputter) error {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case MessageTypeMove:
		if captureRect == nil {
			return nil
		}
		var body MoveMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		x := captureRect.StartX + body.X
		y := captureRect.StartY + body.Y
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

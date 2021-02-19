package gameagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/castaneai/mashimaro/pkg/messaging"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/streamer"
)

var (
	errGameExited = errors.New("game exited")
)

func (a *Agent) startMessaging(ctx context.Context, captureDisplay string, message <-chan []byte, captureAreaChanged <-chan *streamer.CaptureArea) error {
	log.Printf("initialing x11 connection")
	xu, err := xgbutil.NewConnDisplay(captureDisplay)
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
			if err := a.handleMessage(ctx, msg, captureArea, xu, xi); err != nil {
				if err == errGameExited {
					return err
				}
				log.Printf("failed to handle message: %+v", err)
			}
		}
	}
}

func (a *Agent) handleMessage(ctx context.Context, data []byte, captureArea *streamer.CaptureArea, xu *xgbutil.XUtil, xinput *x11.Inputter) error {
	var msg messaging.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case messaging.MessageTypeMove:
		var body messaging.MoveMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		x := captureArea.StartX + body.X
		y := captureArea.StartY + body.Y
		xinput.Move(x, y)
		return nil
	case messaging.MessageTypeMouseDown:
		var body messaging.MouseDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendButton(xu.RootWin(), xproto.Button(body.Button), true)
		return nil
	case messaging.MessageTypeMouseUp:
		var body messaging.MouseUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendButton(xu.RootWin(), xproto.Button(body.Button), false)
		return nil
	case messaging.MessageTypeKeyDown:
		var body messaging.KeyDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendKey(xu.RootWin(), xproto.Keycode(body.Key), true)
		return nil
	case messaging.MessageTypeKeyUp:
		var body messaging.KeyUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xinput.SendKey(xu.RootWin(), xproto.Keycode(body.Key), false)
		return nil
	case messaging.MessageTypeExitGame:
		if _, err := a.gameWrapperClient.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
			return err
		}
		return errGameExited
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

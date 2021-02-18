package gameagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/xgb/xproto"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"

	"github.com/castaneai/mashimaro/pkg/messaging"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
	msgChBufferSize   = 50
)

var (
	ErrGameExited              = errors.New("game exited")
	defaultWebRTCConfiguration = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
)

type Agent struct {
	brokerClient      proto.BrokerClient
	gameWrapperClient proto.GameWrapperClient
	signaler          Signaler
	streamingConfig   *StreamingConfig
	captureArea       streamer.CaptureArea

	onExit     func()
	callbackMu sync.Mutex

	xutil  *xgbutil.XUtil
	xinput *x11.Inputter
}

type StreamingConfig struct {
	XDisplay  string
	PulseAddr string

	X264Params   string
	DisableAudio bool
}

func NewAgent(brokerClient proto.BrokerClient, gameWrapperClient proto.GameWrapperClient, signaler Signaler, streamingConfig *StreamingConfig) *Agent {
	if streamingConfig.X264Params == "" {
		streamingConfig.X264Params = defaultX264Params
	}
	return &Agent{
		brokerClient:      brokerClient,
		gameWrapperClient: gameWrapperClient,
		signaler:          signaler,
		streamingConfig:   streamingConfig,
	}
}

func (a *Agent) OnExit(f func()) {
	a.callbackMu.Lock()
	defer a.callbackMu.Unlock()
	a.onExit = f
}

func (a *Agent) Run(ctx context.Context, gsName string) error {
	log.Printf("waiting for session...")
	ss, err := waitForSession(ctx, a.brokerClient, gsName)
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(ss.SessionId)
	var metadata gamemetadata.Metadata
	if err := yaml.Unmarshal([]byte(ss.GameMetadata.Body), &metadata); err != nil {
		return err
	}
	log.Printf("[agent] load metadata: %+v", metadata)

	log.Printf("[agent] initialing x11 connection")
	xu, err := xgbutil.NewConnDisplay(a.streamingConfig.XDisplay)
	if err != nil {
		return errors.Wrap(err, "failed to connect to X11")
	}
	xi, err := x11.NewInputter(xu)
	if err != nil {
		return errors.Wrap(err, "failed to new X11 inputter")
	}
	a.xutil = xu
	a.xinput = xi

	// TODO: provisioning game data and ready to start process
	log.Printf("--- (TODO) provisioning game...")

	conn, err := transport.NewWebRTCStreamerConn(defaultWebRTCConfiguration)
	if err != nil {
		return errors.Wrap(err, "failed to new webrtc streamer conn")
	}
	connected := make(chan struct{})
	conn.OnConnect(func() {
		close(connected)
	})
	msgCh := make(chan []byte, msgChBufferSize)
	conn.OnMessage(func(data []byte) {
		msgCh <- data
	})

	log.Printf("[agent] start signaling...")
	if err := a.signaler.Signaling(ctx, conn, string(sid), "streamer"); err != nil {
		return err
	}

	// TODO: connection timed out
	log.Printf("[agent] waiting for connection...")
	<-connected
	log.Printf("[agent] connected!")

	if err := a.startGame(ctx, &metadata); err != nil {
		return err
	}

	areaCh := make(chan *streamer.CaptureArea)
	go func() {
		st, err := a.gameWrapperClient.ListenCaptureArea(ctx, &proto.ListenCaptureAreaRequest{})
		if err != nil {
			log.Printf("failed to listen capture area: %+v", err)
			return
		}
		for {
			resp, err := st.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("failed to recv listen capture area: %+v", err)
			}
			areaCh <- &streamer.CaptureArea{
				StartX: int(resp.StartX),
				StartY: int(resp.StartY),
				EndX:   int(resp.EndX),
				EndY:   int(resp.EndY),
			}
		}
	}()

	a.captureArea = *<-areaCh
	xconf := &streamer.X11CaptureConfig{
		Display:     a.streamingConfig.XDisplay,
		CaptureArea: &a.captureArea,
	}
	videoStream, err := streamer.NewX11VideoStream(xconf, a.streamingConfig.X264Params)
	if err != nil {
		return errors.Wrap(err, "failed to get x11 video stream")
	}
	defer videoStream.Close()
	go func() {
		videoStream.Start()
		if err := startSendingVideoToConn(ctx, conn, videoStream); err != nil {
			log.Printf("failed to send video: %+v", err)
		}
	}()
	if !a.streamingConfig.DisableAudio {
		audioStream, err := streamer.NewPulseAudioStream(a.streamingConfig.PulseAddr)
		if err != nil {
			return errors.Wrap(err, "failed to get pulse audio stream")
		}
		defer audioStream.Close()
		go func() {
			audioStream.Start()
			if err := startSendingAudioToConn(ctx, conn, audioStream); err != nil {
				log.Printf("failed to send audio: %+v", err)
			}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgCh:
			if err := a.handleMessage(ctx, msg); err != nil {
				if errors.Is(err, ErrGameExited) {
					a.handleExit()
					return nil
				}
				log.Printf("failed to handle message: %+v", err)
			}
		}
	}
}

func (a *Agent) startGame(ctx context.Context, metadata *gamemetadata.Metadata) error {
	cmds := strings.Split(metadata.Command, " ")
	var args []string
	if len(cmds) > 1 {
		args = cmds[1:]
	}
	if _, err := a.gameWrapperClient.StartGame(ctx, &proto.StartGameRequest{
		Command: cmds[0],
		Args:    args,
	}); err != nil {
		return errors.Wrap(err, "failed to start game")
	}
	go a.startHealthCheck(ctx, 3*time.Second)
	return nil
}

func waitForSession(ctx context.Context, c proto.BrokerClient, gsName string) (*proto.Session, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := c.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gsName})
			if err != nil {
				return nil, errors.Wrap(err, "failed to find session by broker")
			}
			if !resp.Found {
				continue
			}
			var metadata gamemetadata.Metadata
			if err := yaml.Unmarshal([]byte(resp.Session.GameMetadata.Body), &metadata); err != nil {
				return nil, err
			}
			log.Printf("found session(sid: %s, gameId: %s)", resp.Session.SessionId, metadata.GameID)
			return resp.Session, nil
		}
	}
}

func (a *Agent) handleMessage(ctx context.Context, data []byte) error {
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
		x := a.captureArea.StartX + body.X
		y := a.captureArea.StartY + body.Y
		a.xinput.Move(x, y)
		return nil
	case messaging.MessageTypeMouseDown:
		var body messaging.MouseDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		a.xinput.SendButton(a.xutil.RootWin(), xproto.Button(body.Button), true)
		return nil
	case messaging.MessageTypeMouseUp:
		var body messaging.MouseUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		a.xinput.SendButton(a.xutil.RootWin(), xproto.Button(body.Button), false)
		return nil
	case messaging.MessageTypeKeyDown:
		var body messaging.KeyDownMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		a.xinput.SendKey(a.xutil.RootWin(), xproto.Keycode(body.Key), true)
		return nil
	case messaging.MessageTypeKeyUp:
		var body messaging.KeyUpMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		a.xinput.SendKey(a.xutil.RootWin(), xproto.Keycode(body.Key), false)
		return nil
	case messaging.MessageTypeExitGame:
		if _, err := a.gameWrapperClient.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
			return err
		}
		return ErrGameExited
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (a *Agent) handleExit() {
	a.callbackMu.Lock()
	h := a.onExit
	a.callbackMu.Unlock()
	if h != nil {
		h()
	}
}

func (a *Agent) startHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := a.gameWrapperClient.HealthCheck(ctx, &proto.HealthCheckRequest{})
			if err != nil {
				log.Printf("failed to health check: %+v", err)
				return
			}
			if !resp.Healthy {
				log.Printf("game process is unhealthy(exited)")
				return
			}
		}
	}
}

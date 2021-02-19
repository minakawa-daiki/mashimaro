package gamewrapper

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/BurntSushi/xgb/xproto"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgbutil"
	"github.com/castaneai/mashimaro/pkg/x11"
)

type area struct {
	startX int
	startY int
	endX   int
	endY   int
}

func (a *area) Width() int {
	return a.endX - a.startX
}

func (a *area) Height() int {
	return a.endY - a.startY
}

type processWatcher struct {
	cmd   *exec.Cmd
	pid   int
	pidMu sync.Mutex

	area        area
	areaChanged chan area
}

func newProcessWatcher() *processWatcher {
	return &processWatcher{
		areaChanged: make(chan area),
	}
}

func (w *processWatcher) Start(cmd *exec.Cmd) error {
	w.cmd = cmd
	w.pidMu.Lock()
	w.pid = cmd.Process.Pid
	w.pidMu.Unlock()
	xu, err := xgbutil.NewConn()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	log.Printf("start watching process: %+v", w.cmd)
	for {
		select {
		case <-ticker.C:
			if err := w.checkWindows(xu); err != nil {
				return errors.Wrap(err, "failed to check windows")
			}
		}
	}
}

func (w *processWatcher) AreaChanged() <-chan area {
	return w.areaChanged
}

func (w *processWatcher) findWindows(xu *xgbutil.XUtil) ([]xproto.Window, error) {
	w.pidMu.Lock()
	pid := w.pid
	w.pidMu.Unlock()
	if pid == 0 {
		return nil, errors.New("pid is not set; start process first")
	}
	windows, err := x11.EnumWindows(xu, xu.RootWin(), true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enum window")
	}
	return windows, nil
}

func (w *processWatcher) checkWindows(xu *xgbutil.XUtil) error {
	windows, err := w.findWindows(xu)
	if err != nil {
		return err
	}
	if len(windows) == 0 {
		return nil
	}
	mainWindow, err := x11.GetMainWindow(xu, windows)
	if err != nil {
		return errors.Wrap(err, "failed to get main window")
	}
	x, y, err := x11.GetWindowPositionOnScreen(xu, xu.Screen(), mainWindow)
	if err != nil {
		return errors.Wrap(err, "failed to get window position")
	}
	width, height, err := x11.GetWindowSize(xu, mainWindow)
	if err != nil {
		return errors.Wrap(err, "failed to get window size")
	}
	area := &area{
		startX: x,
		startY: y,
		endX:   x + width - 1,
		endY:   y + height - 1,
	}
	if areaIsValid(area) && areaHasChanged(area, &w.area) {
		log.Printf("capture area has changed: (%dx%d)", area.Width(), area.Height())
		w.area = *area
		w.areaChanged <- *area
	}
	return nil
}

func areaIsValid(a *area) bool {
	return a.Width() > 0 && a.Height() > 0
}

func areaHasChanged(a1, a2 *area) bool {
	return a1.startX != a2.startX ||
		a1.startY != a2.startY ||
		a1.endX != a2.endX ||
		a1.endY != a2.endY
}

func (w *processWatcher) KillProcess() error {
	w.pidMu.Lock()
	pid := w.pid
	w.pidMu.Unlock()
	if pid == 0 {
		return errors.New("pid is not set; start process first")
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrapf(err, "failed to find process(pid: %d)", pid)
	}
	if err := p.Signal(os.Interrupt); err != nil {
		return errors.Wrap(err, "failed to interrupt game process")
	}
	return nil
}

func (w *processWatcher) IsLiving() bool {
	// TODO:
	return true
}

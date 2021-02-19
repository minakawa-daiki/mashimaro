package gamewrapper

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/tevino/abool"

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

type processWatcher struct {
	cmd    *exec.Cmd
	pid    int
	pidMu  sync.Mutex
	exited *abool.AtomicBool

	area        area
	areaChanged chan area
}

func newProcessWatcher() *processWatcher {
	return &processWatcher{
		exited:      abool.New(),
		areaChanged: make(chan area),
	}
}

func (w *processWatcher) Start(cmd *exec.Cmd) error {
	w.cmd = cmd
	w.pidMu.Lock()
	w.pid = cmd.Process.Pid
	w.pidMu.Unlock()
	exited := make(chan error)
	go func() {
		exited <- w.cmd.Wait()
	}()
	xu, err := xgbutil.NewConn()
	if err != nil {
		return err
	}
	if err := w.checkWindows(xu); err != nil {
		return errors.Wrap(err, "failed to check windows")
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	log.Printf("start watching process: %+v", w.cmd)
	for {
		select {
		case <-exited:
			w.exited.Set()
			code := w.cmd.ProcessState.ExitCode()
			log.Printf("process exited with code %v", code)
			return nil
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

func (w *processWatcher) checkWindows(xu *xgbutil.XUtil) error {
	w.pidMu.Lock()
	pid := w.pid
	w.pidMu.Unlock()
	if pid == 0 {
		return errors.New("pid is not set; start process first")
	}
	windows, err := x11.EnumWindowsByPid(xu, pid, xu.RootWin(), true)
	if err != nil {
		return errors.Wrap(err, "failed to enum window")
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
	area := area{
		startX: x,
		startY: y,
		endX:   x + width - 1,
		endY:   y + height - 1,
	}
	if areaHasChanged(&area, &w.area) {
		w.area = area
		w.areaChanged <- area
	}
	return nil
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
	return w.exited.IsNotSet()
}

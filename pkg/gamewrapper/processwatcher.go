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

type processWatcher struct {
	cmd    *exec.Cmd
	pid    int
	pidMu  sync.Mutex
	exited *abool.AtomicBool
}

func newProcessWatcher() *processWatcher {
	return &processWatcher{
		exited: abool.New(),
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
	for _, window := range windows {
		if err := x11.CenterWindow(xu, xu.Screen(), window, false); err != nil {
			return errors.Wrap(err, "failed to center window")
		}
	}
	/*
		mainWindow, err := x11.GetMainWindow(xu, windows)
		if err != nil {
			return errors.Wrap(err, "failed to get main window")
		}
	*/
	return nil
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

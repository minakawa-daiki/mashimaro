package x11

import (
	"os/exec"
	"testing"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/pkg/errors"
)

func newDefaultXUtil(t *testing.T) *xgbutil.XUtil {
	xu, err := xgbutil.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	return xu
}

func waitForAnyWindow(xu *xgbutil.XUtil, pid int, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			windows, err := EnumWindowsByPid(xu, pid, xu.RootWin(), false)
			if err != nil {
				return err
			}
			if len(windows) > 0 {
				return nil
			}
		case <-timer.C:
			return errors.New("timed out")
		}
	}
}

func startWineProcess(t *testing.T, args ...string) *exec.Cmd {
	cmd := exec.Command("wine", args...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return cmd
}

func mustReadyWineProcess(t *testing.T, xu *xgbutil.XUtil, args ...string) *exec.Cmd {
	cmd := startWineProcess(t, args...)
	if err := waitForAnyWindow(xu, cmd.Process.Pid, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	return cmd
}

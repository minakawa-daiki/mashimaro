package gamewrapper

import (
	"testing"

	"github.com/castaneai/mashimaro/pkg/x11"

	"golang.org/x/sync/errgroup"
)

func TestWatch(t *testing.T) {
	t.Skip("Only on manual test")

	cmd := x11.StartWineProcess(t, "/home/castaneai/Downloads/rivalgame-1.05-win/rivalgame.exe")
	w := newProcessWatcher()

	var eg errgroup.Group
	eg.Go(func() error {
		return w.Start(cmd)
	})
	area := <-w.AreaChanged()
	t.Logf("area: %v", area)
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}

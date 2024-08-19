package game

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Game struct {
	cmd *exec.Cmd
}

func New(bin string, dir string) *Game {
	g := &Game{cmd: exec.Command(bin)}
	g.cmd.Dir = dir
	return g
}

func (g *Game) Binary() string {
	return g.cmd.Path
}

func (g *Game) Dir() string {
	return g.cmd.Dir
}

func (g *Game) Start() error {
	log.Info("starting game", "binary", g.Binary(), "dir", g.Dir())
	return g.cmd.Start()
}

// Stop gracefully stops the game process or terminates it if it doesn't stop in time.
func (g *Game) Stop() error {
	// is the process still running?
	if g.cmd.Process == nil {
		log.Info("process is not running anymore")
		return nil
	}

	log.Info("stopping game process", "pid", g.cmd.Process.Pid, "binary", g.Binary())

	signals := []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL}

	for _, sig := range signals {
		// send the signal
		if err := g.cmd.Process.Signal(sig); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				return nil
			}
			return errors.Join(fmt.Errorf("failed to send signal %s", sig), err)
		}

		// wait for the process to exit with a timeout of 5 seconds
		done := make(chan error, 1)
		go func() {
			if g.cmd.Process == nil || g.cmd.ProcessState != nil {
				done <- nil
				return
			}
			state, err := g.cmd.Process.Wait()
			if err != nil {
				done <- err
				return
			}
			if state.Exited() {
				done <- nil
				return
			}
		}()

		select {
		case err := <-done:
			if err == nil {
				log.Info("process exited successfully")
				return nil
			}
			return errors.Join(errors.New("failed to wait for process to exit"), err)
		case <-time.After(2 * time.Second):
			continue
		}
	}

	// kill
	log.Warn("killing game process")
	return g.cmd.Process.Kill()
}

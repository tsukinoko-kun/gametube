package game

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

type Game struct {
	cmd *exec.Cmd
}

func New(bin string, dir string) Game {
	g := Game{cmd: exec.Command(bin)}
	g.cmd.Dir = dir
	return g
}

func (g *Game) Start() error {
	return g.cmd.Start()
}

// Stop gracefully stops the game process or terminates it if it doesn't stop in time.
func (g *Game) Stop() error {
	// is the process still running?
	if g.cmd.Process == nil {
		fmt.Println("process is not running anymore")
		return nil
	}

	signals := []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL}

	for _, sig := range signals {
		// send the signal
		if err := g.cmd.Process.Signal(sig); err != nil {
			return errors.Join(fmt.Errorf("failed to send signal %s", sig), err)
		}

		// wait for the process to exit with a timeout of 5 seconds
		done := make(chan error, 1)
		go func() {
			done <- g.cmd.Wait()
		}()

		select {
		case err := <-done:
			return errors.Join(errors.New("failed to wait for process to exit"), err)
		case <-time.After(5 * time.Second):
			continue
		}
	}

	// kill
	return g.cmd.Process.Kill()
}

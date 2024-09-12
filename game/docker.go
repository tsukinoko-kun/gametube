package game

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tsukinoko-kun/gametube/config"
)

type (
	Container struct {
		ImageName     string
		Dockerfile    string
		Process       *os.Process
		ContainerName string
		Log           *strings.Builder
		Config        *config.Game
	}
)

const ubuntuGuiDockerfile = `FROM ubuntu:noble

RUN apt-get update

# Install build tools
RUN apt-get install -y pkg-config libx11-dev libasound2-dev libudev-dev libxcb-render0-dev libxcb-shape0-dev libxcb-xfixes0-dev

# Install X11 and audio libraries
RUN apt-get install -y \
    xvfb \
    x11vnc \
    xdotool \
    wget \
    unzip \
    ffmpeg \
    pulseaudio \
    libgl1-mesa-dri \
    mesa-utils libglu1-mesa-dev freeglut3-dev mesa-common-dev \
    libglew-dev libglfw3-dev libglm-dev libao-dev libmpg123-dev

# Install lightweight window manager and desktop environment
RUN apt-get install -y \
    openbox \
    lxqt

# Set the virtual display resolution and color depth
ENV DISPLAY=:99
ENV RESOLUTION=1920x1080
ENV COLOR_DEPTH=24
`

var (
	fakeEnv = map[string]string{
		"XDG_DATA_HOME":   "/root/.local/share",
		"XDG_STATE_HOME":  "/root/.local/state",
		"XDG_CACHE_HOME":  "/root/.cache",
		"XDG_RUNTIME_DIR": "/root/.local/run",
		"XDG_CONFIG_HOME": "/root/.config",
	}
	ubuntuGuiImageBuilt sync.Once
)

func init() {
	// Build the ubuntu-gui base image at server start time
	ubuntuGuiImageBuilt.Do(func() {
		ctx := context.Background()
		if err := buildUbuntuGuiImage(ctx); err != nil {
			log.Fatalf("Failed to build ubuntu-gui base image: %v", err)
		}
		log.Println("ubuntu-gui base image built successfully")
	})
}

func expandEnv(s string) string {
	return os.Expand(s, func(s string) string {
		return fakeEnv[s]
	})
}

func NewContainer(ctx context.Context, sessionId string, c *config.Game) (*Container, error) {
	container := &Container{
		ImageName:  fmt.Sprintf("gametube-%s-%s", c.Slug, sessionId),
		Dockerfile: "",
		Process:    nil,
		Log:        &strings.Builder{},
		Config:     c,
	}

	if err := container.generateDockerfiles(); err != nil {
		return nil, errors.Join(errors.New("failed to generate Dockerfiles"), err)
	}

	if err := container.build(ctx); err != nil {
		return nil, errors.Join(errors.New("failed to build Docker image"), err)
	}

	if process, err := container.run(ctx); err != nil {
		return nil, errors.Join(errors.New("failed to run Docker container"), err)
	} else {
		container.Process = process
	}

	return container, nil
}

func (c *Container) generateDockerfiles() error {
	tempDir, err := os.MkdirTemp("", "gametube-docker-*")
	if err != nil {
		return errors.Join(errors.New("failed to create temp directory"), err)
	}
	c.Dockerfile = filepath.Join(tempDir, "Dockerfile")

	// Generate Dockerfile
	dockerfileContent := c.newDockerfile()
	if err := os.WriteFile(c.Dockerfile, []byte(dockerfileContent), 0644); err != nil {
		return errors.Join(errors.New("failed to write Dockerfile"), err)
	}

	// Generate entrypoint.sh
	entrypointContent := c.newEntrypoint()
	entrypointPath := filepath.Join(tempDir, "entrypoint.sh")
	if err := os.WriteFile(entrypointPath, []byte(entrypointContent), 0755); err != nil {
		return errors.Join(errors.New("failed to write entrypoint.sh"), err)
	}

	return nil
}

func (c *Container) newDockerfile() string {
	sb := strings.Builder{}

	sb.WriteString("FROM ubuntu-gui\n\n")

	if strings.HasPrefix(c.Config.WorkingDirectory, "/") {
		sb.WriteString(fmt.Sprintf("WORKDIR %s\n", expandEnv(c.Config.WorkingDirectory)))
	} else {
		sb.WriteString(fmt.Sprintf("WORKDIR %s\n", path.Join("/game", expandEnv(c.Config.WorkingDirectory))))
	}

	for k, v := range fakeEnv {
		sb.WriteString(fmt.Sprintf("ENV %s=%s\n", k, v))
	}

	sb.WriteString("COPY entrypoint.sh /entrypoint.sh\n")
	sb.WriteString("RUN chmod +x /entrypoint.sh\n\n")

	sb.WriteString("ENTRYPOINT [\"/entrypoint.sh\"]\n")

	return sb.String()
}

func (c *Container) newEntrypoint() string {
	sb := strings.Builder{}

	sb.WriteString("#!/bin/sh\n\n")

	sb.WriteString("# Start virtual framebuffer\n")
	sb.WriteString("Xvfb :99 -screen 0 1920x1080x24 &\n\n")

	sb.WriteString("# Wait for Xvfb to be ready\n")
	sb.WriteString("while ! xdpyinfo -display :99 >/dev/null 2>&1; do\n")
	sb.WriteString("    echo \"Waiting for Xvfb...\"\n")
	sb.WriteString("    sleep 0.1\n")
	sb.WriteString("done\n\n")

	sb.WriteString("# Start window manager\n")
	sb.WriteString("openbox-session &\n")
	sb.WriteString("sleep 1\n\n")

	sb.WriteString("# Start the game\n")
	sb.WriteString(fmt.Sprintf("cd %s\n", c.Config.WorkingDirectory))
	sb.WriteString(fmt.Sprintf("./%s\n", c.Config.Entrypoint))

	return sb.String()
}

func (c *Container) build(ctx context.Context) error {
	// Build the Docker image
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", c.ImageName, "-f", c.Dockerfile, filepath.Dir(c.Dockerfile))

	// Set up MultiWriter to write to both os.Stdout and c.Log
	multiWriter := io.MultiWriter(os.Stdout, c.Log)
	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	if err := cmd.Run(); err != nil {
		return errors.Join(errors.New("failed to build Docker image"), err)
	}

	return nil
}

func buildUbuntuGuiImage(ctx context.Context) error {
	// Check if the ubuntu-gui image already exists
	cmd := exec.Command("docker", "image", "inspect", "ubuntu-gui")
	if err := cmd.Run(); err == nil {
		// Image already exists, no need to build
		return nil
	}

	// Create a temporary Dockerfile for ubuntu-gui
	tmpfile, err := os.CreateTemp("", "ubuntu-gui.Dockerfile")
	if err != nil {
		return errors.Join(errors.New("failed to create temporary Dockerfile for ubuntu-gui"), err)
	}
	defer os.Remove(tmpfile.Name())

	// Write the Dockerfile content
	if _, err := tmpfile.WriteString(ubuntuGuiDockerfile); err != nil {
		return errors.Join(errors.New("failed to write ubuntu-gui Dockerfile"), err)
	}
	if err := tmpfile.Close(); err != nil {
		return errors.Join(errors.New("failed to close temporary ubuntu-gui Dockerfile"), err)
	}

	// Build the ubuntu-gui Docker image
	cmd = exec.CommandContext(ctx, "docker", "build", "-t", "ubuntu-gui", "-f", tmpfile.Name(), ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Join(errors.New("failed to build ubuntu-gui Docker image"), err)
	}

	return nil
}

func (c *Container) run(ctx context.Context) (*os.Process, error) {
	c.ContainerName = fmt.Sprintf("%s-container", c.ImageName)
	cmd := exec.CommandContext(ctx, "docker", "run", "--name", c.ContainerName,
		"-v", fmt.Sprintf("%s:/game:ro", c.Config.Source),
		c.ImageName)

	c.Log.WriteString(fmt.Sprintf("Running command: %v\n", cmd.Args))
	fmt.Printf("Running command: %v\n", cmd.Args)

	// Set up MultiWriter to write to both os.Stdout and c.Log
	multiWriter := io.MultiWriter(os.Stdout, c.Log)
	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	if err := cmd.Start(); err != nil {
		return nil, errors.Join(errors.New("failed to start Docker container"), err)
	}

	// Wait for the container to start
	time.Sleep(2 * time.Second)

	// Get the process ID of the container
	pidCmd := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", c.ContainerName)
	pidOutput, err := pidCmd.Output()
	if err != nil {
		return nil, errors.Join(errors.New("failed to get container PID"), err)
	}

	// Log the output for debugging purposes
	c.Log.WriteString(string(pidOutput))

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidOutput)))
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse container PID"), err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, errors.Join(errors.New("failed to find container process"), err)
	}

	// Start a goroutine to wait for the command to finish and capture its output
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.Log.WriteString(fmt.Sprintf("Container error: %v\n", err))
		}
		c.Log.WriteString(fmt.Sprintf("Container output: %s\n", string(output)))

		if len(output) == 0 {
			c.Log.WriteString("Container output is empty. Checking container logs...\n")
			logCmd := exec.Command("docker", "logs", c.ContainerName)
			logOutput, logErr := logCmd.CombinedOutput()
			if logErr != nil {
				c.Log.WriteString(fmt.Sprintf("Error getting container logs: %v\n", logErr))
			} else {
				c.Log.WriteString(fmt.Sprintf("Container logs: %s\n", string(logOutput)))
			}
		}
	}()

	return process, nil
}

func (c *Container) Close() error {
	// Stop the container
	stopCmd := exec.Command("docker", "stop", c.ContainerName)
	if err := stopCmd.Run(); err != nil {
		// If stopping fails, try to kill the container
		killCmd := exec.Command("docker", "kill", c.ContainerName)
		if killErr := killCmd.Run(); killErr != nil {
			return errors.Join(errors.New("failed to stop or kill container"), err, killErr)
		}
	}

	// Wait for the container to stop (5 seconds timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	waitCmd := exec.CommandContext(ctx, "docker", "wait", c.ContainerName)
	if err := waitCmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			// If timeout occurs, forcefully kill the container
			killCmd := exec.Command("docker", "kill", c.ContainerName)
			if killErr := killCmd.Run(); killErr != nil {
				return errors.Join(errors.New("failed to kill container after timeout"), err, killErr)
			}
		} else {
			return errors.Join(errors.New("error waiting for container to stop"), err)
		}
	}

	// Remove the container
	rmCmd := exec.Command("docker", "rm", c.ContainerName)
	if err := rmCmd.Run(); err != nil {
		return errors.Join(errors.New("failed to remove container"), err)
	}

	// Delete the image
	rmiCmd := exec.Command("docker", "rmi", c.ImageName)
	if err := rmiCmd.Run(); err != nil {
		return errors.Join(errors.New("failed to delete image"), err)
	}

	return nil
}

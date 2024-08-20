FROM golang:latest as build
WORKDIR /gametube
COPY go.mod /gametube/go.mod
COPY go.sum /gametube/go.sum
COPY cmd /gametube/cmd
COPY internal /gametube/internal
COPY static /gametube/static
RUN apt-get update && apt-get install -y libx264-dev
RUN go build -o /gametube/bin/host ./cmd/host

# Start with Ubuntu as the base image
FROM ubuntu:latest AS ubuntu-gui

# Avoid prompts from apt
ENV DEBIAN_FRONTEND=noninteractive

# Update
RUN apt-get update

# Install basic utilities
RUN apt-get install -y \
    wget \
    curl \
    ca-certificates

# Install build tools
RUN apt-get install -y pkg-config libx11-dev libasound2-dev libudev-dev libxcb-render0-dev libxcb-shape0-dev libxcb-xfixes0-dev libx264-dev

# Install X11 and audio libraries
RUN apt-get install -y \
    xvfb \
    x11vnc \
    xdotool \
    wget \
    unzip \
    # FFmpeg for video/audio capture and streaming
    ffmpeg \
    # PulseAudio for audio support
    pulseaudio \
    # OpenGL libraries for 3D acceleration
    libgl1-mesa-dri \
    mesa-utils libglu1-mesa-dev freeglut3-dev mesa-common-dev \
    libglew-dev libglfw3-dev libglm-dev libao-dev libmpg123-dev

# Install lightweight window manager and desktop environment
RUN apt-get install -y \
    # Openbox as a lightweight window manager
    openbox \
    # LXQt as a lightweight desktop environment
    lxqt

# Set the virtual display resolution and color depth
ENV DISPLAY=:99
ENV RESOLUTION=1920x1080
ENV COLOR_DEPTH=24

FROM ubuntu-gui AS gametube

# Set up the entrypoint
COPY --from=build /gametube/bin/host /gametube/host
COPY entrypoint.sh /gametube/entrypoint.sh
RUN chmod +x /gametube/entrypoint.sh

# Start gametube
ENTRYPOINT ["/gametube/entrypoint.sh"]

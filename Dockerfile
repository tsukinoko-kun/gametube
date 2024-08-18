FROM golang:latest as build
WORKDIR /gametube
COPY go.mod /gametube/go.mod
COPY go.sum /gametube/go.sum
RUN go build -o /gametube/bin/host ./cmd/host

# Start with Ubuntu as the base image
FROM ubuntu:22.04 AS base

# Set up the entrypoint
COPY --from=build /gametube/bin/host /gametube/host
COPY entrypoint.sh /gametube/entrypoint.sh
RUN chmod +x /gametube/entrypoint.sh

# Avoid prompts from apt
ENV DEBIAN_FRONTEND=noninteractive

# Update and install basic utilities
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install X11 and audio libraries
RUN apt-get update && apt-get install -y \
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
    libgl1-mesa-glx \
    libgl1-mesa-dri \
    && rm -rf /var/lib/apt/lists/*

# Install lightweight window manager and desktop environment
RUN apt-get update && apt-get install -y \
    # Openbox as a lightweight window manager
    openbox \
    # LXQt as a lightweight desktop environment
    lxqt \
    && rm -rf /var/lib/apt/lists/*

# Set the virtual display resolution and color depth
ENV DISPLAY=:99
ENV RESOLUTION=1920x1080
ENV COLOR_DEPTH=24

FROM base AS gametube

# Start gametube
ENTRYPOINT ["/gametube/entrypoint.sh"]

# Start with Ubuntu as the base image
FROM ubuntu:22.04 AS base

# Avoid prompts from apt
ENV DEBIAN_FRONTEND=noninteractive

# Update and install basic utilities
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install X11 and audio libraries
FROM base AS x11
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
FROM x11 AS gui
RUN apt-get update && apt-get install -y \
    # Openbox as a lightweight window manager
    openbox \
    # LXQt as a lightweight desktop environment
    lxqt \
    && rm -rf /var/lib/apt/lists/*

# Install Rust and required dependencies
FROM gui AS rust

RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libx11-dev \
    libxext-dev \
    libxft-dev \
    libxinerama-dev \
    libxcursor-dev \
    libxrender-dev \
    libxfixes-dev \
    libxdo-dev \
    libssl-dev \
    ffmpeg

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Set up the working directory
WORKDIR /gametube

# Copy your Rust project files (assuming they're in the same directory as the Dockerfile)
COPY Cargo.toml ./Cargo.toml
COPY Cargo.lock ./Cargo.lock
COPY src ./src

# Build the Rust project
RUN cargo build --release

FROM rust AS gametube

WORKDIR /gametube

# Set up the entrypoint
COPY entrypoint.sh /gametube/entrypoint.sh
RUN chmod +x /gametube/entrypoint.sh

# Set the virtual display resolution and color depth
ENV DISPLAY=:99
ENV RESOLUTION=1920x1080
ENV COLOR_DEPTH=24

# Set the entrypoint
ENTRYPOINT ["/gametube/entrypoint.sh"]

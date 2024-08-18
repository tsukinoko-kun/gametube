#!/bin/sh

# Start virtual framebuffer
rm /tmp/.X99-lock > /dev/null 2>&1
Xvfb :99 -screen 0 "${RESOLUTION}x${COLOR_DEPTH}" &

# Wait for Xvfb to be ready
while ! xdpyinfo -display :99 >/dev/null 2>&1; do
    echo "Waiting for Xvfb..."
    sleep 0.1
done

# Start window manager
openbox-session &
sleep 1

# Start your Rust application
/gametube/host /game/mageanoid

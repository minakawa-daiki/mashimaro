#!/bin/bash
echo "Waiting for X socket"
until [[ -S /tmp/.X11-unix/X${DISPLAY/:/} ]]; do sleep 1; done
echo "X socket is ready"

# Run xvfb to avoid error "Failed to query current display settings..."
Xvfb "${DISPLAY}" -screen 0 1280x960x24 &
x11vnc -display WAIT"${DISPLAY}" -shared -forever -passwd 1234 -q &
"$@"

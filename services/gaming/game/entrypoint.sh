#!/bin/sh
# Run xvfb to avoid error "Failed to query current display settings..."
Xvfb "${DISPLAY}" -screen 0 800x600x24 &
x11vnc -display WAIT"${DISPLAY}" -shared -forever -passwd 1234 -q &
"$@"

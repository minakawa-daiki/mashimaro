#!/bin/bash
Xvfb "${DISPLAY}" -screen 0 1920x1080x24 &
x11vnc -display WAIT"${DISPLAY}" -shared -forever -passwd 1234 -q &
"$@"

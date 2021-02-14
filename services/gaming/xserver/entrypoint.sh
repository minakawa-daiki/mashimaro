#!/bin/bash
Xvfb "${DISPLAY}" -screen 0 800x600x24 &
x11vnc -display WAIT"${DISPLAY}" -shared -forever -passwd 1234 -q &

# Wait for X11 to start
echo "Waiting for X socket"
until [[ -S /tmp/.X11-unix/X${DISPLAY/:/} ]]; do sleep 1; done
echo "X socket is ready"

echo "Waiting for X11 startup"
until xhost + >/dev/null 2>&1; do sleep 1; done
echo "X11 startup complete"

# Notify sidecar containers
touch /var/run/appconfig/xserver_ready

# Foreground process, tail logs
tail -n 1000 -F /var/log/Xorg."${DISPLAY/:/}".log
#!/bin/bash

# Start dbus
rm -rf /var/run/dbus
dbus-uuidgen | tee /var/lib/dbus/machine-id
mkdir -p /var/run/dbus
dbus-daemon --config-file=/usr/share/dbus-1/system.conf --print-address

echo "Starting X11 server with Xdummy video driver."
nohup Xorg "${DISPLAY}" -noreset -dpi 96 -novtswitch +extension MIT-SHM +extension GLX +extension RANDR +extension RENDER -config /etc/X11/xorg.conf &
nohup x11vnc -forever &

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

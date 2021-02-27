#!/bin/bash
set +x
echo "Waiting for X server"
until [[ -e /var/run/appconfig/xserver_ready ]]; do sleep 1; done
echo "X server is ready"
set -x
while :; do sleep 3600; done

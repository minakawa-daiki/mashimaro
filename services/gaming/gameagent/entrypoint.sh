#!/bin/bash
set -ex
echo "Waiting for X socket"
until [[ -S /tmp/.X11-unix/X${DISPLAY/:/} ]]; do sleep 1; done
echo "X socket is ready"
"$@"
#!/bin/sh
pulseaudio --daemon --exit-idle-time=-1
while :; do sleep 10; done
# $@

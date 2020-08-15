#!/bin/sh
Xvfb $DISPLAY -screen 0 800x600x24 &
x11vnc -display WAIT:0 -shared -forever -passwd 1234 -q &
$@

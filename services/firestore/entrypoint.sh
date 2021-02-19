#!/bin/sh
nohup gcloud beta emulators firestore start --host-port=0.0.0.0:"${PORT}" &
FIRESTORE_EMULATOR_HOST=localhost:"${PORT}" /seeder "$@"
while :; do sleep 3600; done

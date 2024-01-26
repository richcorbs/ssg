#!/bin/bash

SRC_DIR="./src"
DIST_DIR="./dist"

# Make sure there is a built site
./build.sh

# Watch for changes in the source directory and trigger a build
fswatch -r "$SRC_DIR" |
while read -r changed_file; do
    echo "Detected change in $changed_file. Rebuilding..."
    ./build.sh
done &

# Serve the site using browser-sync
browser-sync start --server "$DIST_DIR" --files "$DIST_DIR/**/*"

#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
fbcdir="$workspace/src/github.com/fairblock"
if [ ! -L "$fbcdir/go-fairblock" ]; then
    mkdir -p "$fbcdir"
    cd "$fbcdir"
    ln -s ../../../../../. go-fairblock
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$fbcdir/go-fairblock"
PWD="$fbcdir/go-fairblock"

# Launch the arguments with the configured environment.
exec "$@"

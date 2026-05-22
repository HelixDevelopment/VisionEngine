#!/bin/bash
# Push to all remotes
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$DIR/push-vasic-digital-github.sh"
"$DIR/push-vasic-digital-gitlab.sh"
"$DIR/push-helix-github.sh"
"$DIR/push-helix-gitlab.sh"
echo "Pushed to all 4 remotes"

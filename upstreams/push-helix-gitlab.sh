#!/bin/bash
# Push to HelixDevelopment GitLab
set -e
git push helix-gitlab master --tags
echo "Pushed to HelixDevelopment/VisionEngine on GitLab"

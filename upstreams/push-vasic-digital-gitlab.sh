#!/bin/bash
# Push to vasic-digital GitLab
set -e
git push vasic-digital-gitlab master --tags
echo "Pushed to vasic-digital/VisionEngine on GitLab"

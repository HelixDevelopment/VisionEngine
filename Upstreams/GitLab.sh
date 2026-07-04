#!/bin/bash
#
# GitLab.sh - GitLab upstream repository for VisionEngine (vasic-digital org).
#
# Sourced by install_upstreams.sh, which reads UPSTREAMABLE_REPOSITORY and
# configures the `GitLab` git remote. Per §6.W, own-org submodules mirror to
# GitHub + GitLab only (CLI parity: gh + glab).

export UPSTREAMABLE_REPOSITORY="git@gitlab.com:vasic-digital/visionengine.git"

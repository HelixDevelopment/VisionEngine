#!/bin/bash
#
# GitHub.sh - GitHub upstream repository for VisionEngine (vasic-digital org).
#
# Sourced by install_upstreams.sh, which reads UPSTREAMABLE_REPOSITORY and
# configures the `GitHub` git remote. Per §6.W, own-org submodules mirror to
# GitHub + GitLab only (CLI parity: gh + glab).

export UPSTREAMABLE_REPOSITORY="git@github.com:vasic-digital/VisionEngine.git"

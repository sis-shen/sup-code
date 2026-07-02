#!/usr/bin/env sh
# entrypoint.sh — Docker 容器入口
set -e

exec supcode "$@"

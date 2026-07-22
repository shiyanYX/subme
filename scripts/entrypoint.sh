#!/bin/sh
set -e

# Copy default collectors to volume if the directory is empty
if [ -d /app/collectors ] && [ -z "$(ls -A /app/collectors)" ]; then
  echo "Collectors directory is empty, populating defaults..."
  cp -r /app/default-collectors/* /app/collectors/
fi

exec subme "$@"

#!/usr/bin/env bash

set -euo pipefail

GOOS="linux" go build -ldflags='-s -w' -tags osusergo -o bin/main github.com/paketo-buildpacks/maven/cmd/main

if [ "${STRIP:-false}" != "false" ]; then
  strip bin/main
fi

if [ "${COMPRESS:-false}" != "false" ]; then
  upx -q -9 bin/main
fi

ln -fs main bin/build
ln -fs main bin/detect

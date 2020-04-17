#!/usr/bin/env bash

set -euo pipefail

if [[ -d ../go-cache ]]; then
  GOPATH=$(realpath ../go-cache)
  export GOPATH
fi

GOOS="linux" go build -ldflags='-s -w' -tags osusergo -o bin/build github.com/paketo-buildpacks/maven/cmd/build
GOOS="linux" go build -ldflags='-s -w' -tags osusergo -o bin/detect github.com/paketo-buildpacks/maven/cmd/detect

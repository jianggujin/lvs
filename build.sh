#!/bin/bash

set -e

echo "[ BUILD RELEASE ]"
BIN_DIR=$(pwd)/bin/
SUFFIX=""
APP_NAME="lvs"
BuildVersion="v1.0.0"
BuildTime=$(go run time.go)

# -ldflag 参数
GOLDFLAGS="-X 'jianggujin.com/lvs/internal/config.BuildTime=$BuildTime'"
GOLDFLAGS+=" -X 'jianggujin.com/lvs/internal/config.BuildVersion=$BuildVersion'"

rm -rf "$BIN_DIR"
mkdir -p "$BIN_DIR"

dist() {
    echo "[ TRY BUILD GOOS=$1 GOARCH=$2 ]"
    export GOOS=$g
    export GOARCH=$a
    export CGO_ENABLED=0
    if [ "$1" == "windows" ]; then
      SUFFIX=".exe"
    else
      SUFFIX=""
    fi
    go build -v -trimpath -ldflags "$GOLDFLAGS" -o "$BIN_DIR/$APP_NAME-$1-$2$SUFFIX" "./cmd"
    unset GOOS
    unset GOARCH
    echo "[ BUILD SUCCESS GOOS=$1 GOARCH=$2 ]"
}

if [ "$1" == "dist" ]; then
    echo "[ DIST ALL PLATFORM ]"
    for g in "windows" "linux" "darwin"; do
        dist "$g" "amd64"
    done
    dist "darwin" "arm64"
else
  # build the current platform
  export GOOS=$(go env get GOOS | sed ':a;N;$!ba;s/^\n*//;s/\n*$//')
  export GOARCH=$(go env get GOARCH | sed ':a;N;$!ba;s/^\n*//;s/\n*$//')
  echo "[ DIST CURRENT PLATFORM GOOS=$GOOS GOARCH=$GOARCH ]"
  if [ "$GOOS" == "windows" ]; then
    SUFFIX=".exe"
  fi
  go build -v -trimpath -ldflags "$GOLDFLAGS" -o "$BIN_DIR/$APP_NAME$SUFFIX" "./cmd"
  echo "[ BUILD SUCCESS GOOS=$GOOS GOARCH=$GOARCH ]"
fi


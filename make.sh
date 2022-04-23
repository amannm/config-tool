#!/usr/bin/env bash

PROJECT_NAME="configism"
PROJECT_MODULE="github.com/amannm/${PROJECT_NAME}"

PROJECT_ROOT="${PWD}"
VERSION=$(< "${PROJECT_ROOT}/version.txt")
BUILD_DIR="${PROJECT_ROOT}/bin"
DIST_DIR="${PROJECT_ROOT}/dist"
SOURCE_DIR="${PROJECT_ROOT}/cmd/${PROJECT_NAME}"

BINARY_PATH="${BUILD_DIR}/${PROJECT_NAME}"

function build {
  mkdir -p "${BUILD_DIR}"
  go build -trimpath -ldflags "-X ${PROJECT_MODULE}/internal/build.Version=${VERSION}" -o "${BINARY_PATH}" "${SOURCE_DIR}"
  chmod +x "${bin_path}"
}

function dist {
  local os
  local arch
  os=$1
  arch=$2

  mkdir -p "${BUILD_DIR}"
  GOOS="${os}" GOOARCH="${arch}" go build -trimpath -ldflags "-X ${PROJECT_MODULE}/internal/build.Version=${VERSION}" -o "${BINARY_PATH}" "${SOURCE_DIR}"
  mkdir -p "${DIST_DIR}"
  tar -czvf "${DIST_DIR}/${PROJECT_NAME}_v${VERSION}_${os}_${arch}.tar.gz" "${BINARY_PATH}"
  rm -rf "${BUILD_DIR}"
}

function prepare {
  go fmt ./...
  golangci-lint run -v ./...
}

function test {
  go test -v ./...
}

function clean {
  rm -rf "${BUILD_DIR}"
  rm -rf "${DIST_DIR}"
}

$@
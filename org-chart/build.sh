#!/bin/sh

mkdir -p dist
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ./dist/ org-chart.go
GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ./dist/ org-chart.go

if ! command -v upx > /dev/null 2>&1
then
  upx --best --lzma ./dist/org-chart ./dist/org-chart || true
fi

echo "Compilation completed, please find executables in dist directory."
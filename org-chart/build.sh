#!/bin/sh

mkdir -p dist
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o ./dist/org-chart org-chart.go && upx ./dist/org-chart
GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -ldflags "-s -w" -o ./dist/org-chart.exe org-chart.go && upx ./dist/org-chart.exe

echo "Compilation completed, please find executables in dist directory."
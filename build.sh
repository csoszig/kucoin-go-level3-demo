#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o kucoin-level3-for-linux cmd/main.go

CGO_ENABLED=0 go build -ldflags '-s -w' -o kucoin-level3-for-mac cmd/main.go

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o kucoin-level3-for-windows.exe cmd/main.go

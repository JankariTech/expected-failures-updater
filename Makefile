SHELL=/bin/bash

.PHONY: build
build: go.mod
	go build -o ocBddKit main.go

SHELL=cmd.exe
.POHONY: build tests

build:
	go build -o pbmanager.exe

tests:
	go test ./...
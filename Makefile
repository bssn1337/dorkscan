.PHONY: build linux windows tidy clean

build:
	go build -o dorkscan .

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/dorkscan-linux-amd64 .

linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/dorkscan-linux-arm64 .

windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/dorkscan-windows-amd64.exe .

tidy:
	go mod tidy

clean:
	rm -rf dist/ dorkscan dorkscan.exe

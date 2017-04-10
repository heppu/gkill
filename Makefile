package = github.com/heppu/gkill

.PHONY: release

release:
	mkdir -p release
	go get -u
	GOOS=darwin GOARCH=amd64 go build -o release/gkill-darwin-amd64 $(package)
	GOOS=linux GOARCH=amd64 go build -o release/gkill-linux-amd64 $(package)
	GOOS=linux GOARCH=386 go build -o release/gkill-linux-386 $(package)
	GOOS=linux GOARCH=arm64 go build -o release/gkill-linux-arm64 $(package)
	GOARM=7 GOOS=linux GOARCH=arm go build -o release/gkill-linux-arm7 $(package)
	GOARM=6 GOOS=linux GOARCH=arm go build -o release/gkill-linux-arm6 $(package)
	GOARM=5 GOOS=linux GOARCH=arm go build -o release/gkill-linux-arm5 $(package)
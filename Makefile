check:
	go test -cover -v github.com/espebra/filebin/app/api github.com/espebra/filebin/app/model github.com/espebra/filebin/app/config

get-deps:
	go get github.com/dustin/go-humanize
	go get github.com/golang/glog
	go get github.com/gorilla/mux

build:
	go build -ldflags "-X main.buildstamp \"$(date -u '+%Y-%m-%d %H:%M:%S')\" -X main.githash $(git rev-parse HEAD)"

install:
	go install -ldflags "-X main.buildstamp \"$(date -u '+%Y-%m-%d %H:%M:%S')\" -X main.githash $(git rev-parse HEAD)"

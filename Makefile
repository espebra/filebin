HASH=`git rev-parse HEAD`
TIME=`date -u '+%Y-%m-%d %H:%M:%S'`

check:
	go test -cover -v github.com/espebra/filebin/app/api github.com/espebra/filebin/app/model github.com/espebra/filebin/app/config

get-deps:
	go get github.com/dustin/go-humanize
	go get github.com/golang/glog
	go get github.com/gorilla/mux
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/elazarl/go-bindata-assetfs/...
	go get github.com/arschles/go-bindata-html-template

build:
	go-bindata-assetfs static/...
	go-bindata -o app/output/templates.go -pkg output templates/...
	
	go build -ldflags "-X main.buildstamp \"${TIME}\" -X main.githash \"${HASH}\""

install:
	go-bindata-assetfs static/...
	go-bindata -o app/output/templates.go -pkg output templates/...
	go install -ldflags "-X main.buildstamp \"${TIME}\" -X main.githash \"${HASH}\""

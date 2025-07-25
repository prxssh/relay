SRC_DIR := cmd/relay
BINARY := relay
BUILD_DIR := build

.PHONY: clean format test

build: 
	mkdir -p ${BUILD_DIR}
	go build -o ${BUILD_DIR}/${BINARY} ${SRC_DIR}/main.go

run: 
	go run ${SRC_DIR}/main.go

clean: 
	go clean 
	rm -rf build

format: 
	golines -m 80 -t 8 -w .
	gofmt -w .

test: 
	go test ./...


# Variables
APP_NAME := rpi-provisioner
BUILD_DIR := ./build

# Build target
build:
	mkdir -p ${BUILD_DIR}
	GOOS=windows GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME}-windows-amd64.exe
	GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME}-linux-amd64
	GOOS=linux GOARCH=arm go build -o ${BUILD_DIR}/${APP_NAME}-linux-arm
	GOOS=darwin GOARCH=amd64 go build -o ${BUILD_DIR}/${APP_NAME}-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o ${BUILD_DIR}/${APP_NAME}-darwin-arm64

# Clean target
clean:
	go clean
	rm -rf ${BUILD_DIR}

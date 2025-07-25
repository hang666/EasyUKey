.PHONY: all server client clean

BUILD_DIR := build

SERVER_GO_ENV := GOOS=linux GOARCH=amd64
CLIENT_WINDOWS_GO_ENV := GOOS=windows GOARCH=amd64
CLIENT_LINUX_GO_ENV := GOOS=linux GOARCH=amd64

ENCRYPT_KEY_STR := 123456789
SERVER_ADDR := http://localhost:8888
DEV_MODE := false

CLIENT_LDFLAGS := -X 'main.EncryptKeyStr=$(ENCRYPT_KEY_STR)' -X 'main.ServerAddr=$(SERVER_ADDR)' -X 'main.DevMode=$(DEV_MODE)'

all: server client client-linux

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

server: $(BUILD_DIR)
	cd server && $(SERVER_GO_ENV) go build -o ../$(BUILD_DIR)/easyukey-server -trimpath -ldflags "-w -s -buildid=" .

client: $(BUILD_DIR)
	cd client && $(CLIENT_WINDOWS_GO_ENV) go build -o ../$(BUILD_DIR)/easyukey-client.exe -trimpath -ldflags "$(CLIENT_LDFLAGS) -w -s -buildid=" .

client-linux: $(BUILD_DIR)
	cd client && $(CLIENT_LINUX_GO_ENV) go build -o ../$(BUILD_DIR)/easyukey-client -trimpath -ldflags "$(CLIENT_LDFLAGS) -w -s -buildid=" .

clean:
	rm -rf $(BUILD_DIR)

.PHONY: all server client clean

BUILD_DIR := build

SERVER_GO_ENV := GOOS=linux GOARCH=amd64
CLIENT_GO_ENV := GOOS=windows GOARCH=amd64

ENCRYPT_KEY_STR := 123456789
SERVER_ADDR := http://localhost:8888
DEV_MODE := false

CLIENT_LDFLAGS := -X 'main.EncryptKeyStr=$(ENCRYPT_KEY_STR)' -X 'main.ServerAddr=$(SERVER_ADDR)' -X 'main.DevMode=$(DEV_MODE)'

all: server client

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

server: $(BUILD_DIR)
	cd server && $(SERVER_GO_ENV) go build -o ../$(BUILD_DIR)/easyukey-server .

client: $(BUILD_DIR)
	cd client && $(CLIENT_GO_ENV) go build -o ../$(BUILD_DIR)/easyukey-client.exe -ldflags "$(CLIENT_LDFLAGS)" .

clean:
	rm -rf $(BUILD_DIR)

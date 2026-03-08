.PHONY: demo build clean

# The output binary name for the hotreload tool
BIN_NAME=hotreload.exe

# The test server output binary name
TEST_BIN=testserver/bin/server.exe

# Build the hotreload CLI
build:
	go build -o bin/$(BIN_NAME) ./cmd/hotreload

# Run the demo using the local hotreload source against the test server
demo:
	go run ./cmd/hotreload --root ./testserver --build "go build -o $(TEST_BIN) ./testserver/main.go" --exec "./$(TEST_BIN)"

# Clean the generated binaries
clean:
	@if exist bin\$(BIN_NAME) del /Q bin\$(BIN_NAME)
	@if exist $(TEST_BIN) del /Q $(TEST_BIN)

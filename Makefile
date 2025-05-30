BINARY_NAME=pbmconnecttlsSynchSample.exe
BINARY_NAME_TEST_HTTP=httpsample.exe
BINARY_NAME_TEST_ASYNCHFLOW=asynchsample.exe


hello:
	echo "Hello"

update: 
	echo "refreshing libraries"
	go get -u all

test:
	echo "running tests..."
	go test -v -race ./pkg/...

test-coverage:
	echo "running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./pkg/...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

build:
	echo "building..."
	go mod tidy	
	go build ./...

run:
#   go build -o ${BINARY_NAME} cmd/examplePBM/main.go
	go build -o ${BINARY_NAME_TEST_HTTP} cmd/httpPBM/main.go
#	go build -o ${BINARY_NAME_TEST_ASYNCHFLOW} cmd/asynchflowpbm/main.go
#	./${BINARY_NAME}
	./${BINARY_NAME_TEST_HTTP}
#	./${BINARY_NAME_TEST_ASYNCHFLOW}

	
clean: 
	go clean
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME_TEST_HTTP}
	rm -f ${BINARY_NAME_TEST_ASYNCHFLOW}
	rm -f coverage.out coverage.html
	rm -rf bin/

ci: test build
	echo "CI pipeline completed successfully"

.PHONY: hello update test test-coverage build run clean ci


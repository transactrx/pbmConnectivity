BINARY_NAME=pbmconnecttlsSynchSample.exe
BINARY_NAME_TEST_HTTP=httpsample.exe
BINARY_NAME_TEST_ASYNCHFLOW=asynchsample.exe


hello:
	echo "Hello"

update: 
	echo "refreshing libraries"
	go get -u all

build:
	echo "building..."
	go mod tidy	
	go build ./...

run:
#   go build -o ${BINARY_NAME} cmd/examplePBM/main.go
#	go build -o ${BINARY_NAME_TEST_HTTP} cmd/httpPBM/main.go
	go build -o ${BINARY_NAME_TEST_ASYNCHFLOW} cmd/asynchflowpbm/main.go
#	./${BINARY_NAME}
#	./${BINARY_NAME_TEST_HTTP}
	./${BINARY_NAME_TEST_ASYNCHFLOW}

	
clean: 
	go clean
	rm ${BINARY_NAME}
	rm ${BINARY_NAME_TEST_HTTP}
	rm ${BINARY_NAME_TEST_ASYNCHFLOW}


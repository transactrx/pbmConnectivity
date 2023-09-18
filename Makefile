BINARY_NAME=pbmconnectivity.exe

hello:
	echo "Hello"

build:
	echo "building..."
	go mod tidy	
	go build ./...

run:
	go build -o ${BINARY_NAME} cmd/main/main.go
	./${BINARY_NAME}
	

clean: 
	go clean
	rm ${BINARY_NAME}


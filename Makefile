BINARY_NAME=pbmconnecttlsSynchSample.exe




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
	go build -o ${BINARY_NAME} cmd/examplePBM/main.go
	./${BINARY_NAME}
	

clean: 
	go clean
	rm ${BINARY_NAME}


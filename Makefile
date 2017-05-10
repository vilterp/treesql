all: treesql-client treesql-server

start:
	go run server/server.go

deps:
	godep restore
	cd webui && npm install

treesql-client:
	godep go build -o treesql-client client/client.go

treesql-server:
	godep go build -o treesql-server server/server.go

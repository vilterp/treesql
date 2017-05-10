all: treesql-client treesql-server

treesql-client:
	godep go build -o treesql-client client/client.go

treesql-server:
	godep go build -o treesql-server server/server.go

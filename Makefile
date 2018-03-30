all: treesql-server webui treesql-shell

start:
	go run server/server.go

start-dev-server:
	cd webui && PORT=9001 npm run start

deps:
	godep restore
	cd webui && npm install

webui:
	cd webui && npm run build

treesql-server:
	godep go build -v -o treesql-server cmd/server/server.go

treesql-shell:
	godep go build -v -o treesql-shell cmd/shell/shell.go

clean:
	rm -r treesql-server
	rm -r webui/build

.PHONY: webui treesql-server-linux treesql-server treesql-shell test

test:
	go test ./... -timeout 10s

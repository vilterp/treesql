all: treesql-server treesql-server-linux webui treesql-shell

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
	godep go build -o treesql-server server/server.go

treesql-server-linux:
	GOOS=linux godep go build -o treesql-server-linux server/server.go

treesql-shell:
	godep go build -o treesql-shell shell/shell.go

clean:
	rm -r treesql-server
	rm -r webui/build

.PHONY: webui treesql-server-linux treesql-server treesql-shell test

test:
	go test ./...

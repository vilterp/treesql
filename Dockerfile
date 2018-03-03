FROM golang:1.9

RUN go get github.com/tools/godep

RUN mkdir -p $GOPATH/src/github.com/vilterp/treesql
WORKDIR $GOPATH/src/github.com/vilterp/treesql
RUN pwd
COPY . .
RUN godep restore

RUN make treesql-server

EXPOSE 9000
VOLUME ["/data"]

CMD /go/src/github.com/vilterp/treesql/treesql-server --data-file /data/treesql.boltdb

FROM golang:1.12

WORKDIR /src

COPY . .
RUN go mod download

RUN make treesql-server
RUN make test

EXPOSE 9000
VOLUME ["/data"]

CMD ./treesql-server --data-file /data/treesql.boltdb

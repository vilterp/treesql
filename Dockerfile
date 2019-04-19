FROM golang:1.12

WORKDIR /src

COPY . .
RUN make deps

RUN make treesql-server

EXPOSE 9000
VOLUME ["/data"]

CMD ./treesql-server --data-file /data/treesql.boltdb

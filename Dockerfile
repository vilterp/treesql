FROM avcosystems/golang-node:1.13.0

WORKDIR /src

COPY . .
RUN make deps
RUN make all

EXPOSE 9000
VOLUME ["/data"]

CMD ./treesql-server --data-file /data/treesql.boltdb

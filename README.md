# TreeSQL

A simple database to explore alternative query languages which return trees of data instead of tables (hence the name).

Currently uses [BoltDB](https://github.com/boltdb/bolt) as a storage layer, which means it can't do concurrent write transactions.

## Requirements

- Go 1.8
- `go get`ing a bunch of stuff (idk how you are supposed to manage deps in go)

## Usage

`cd` into this directory, then:

Start the server:

```
$ go run server/server.go --port 9000 --data-dir data
```

Start the client:

```
$ go run client/client.go --port 9000
```

And type in queries like

```
many blog_posts { title, comments: many comments { body } }
```

Currently the whole query has to be on one line. Also, `\d` lists tables, and `\d <tableName>` shows the schema of a table (`psql`-style).

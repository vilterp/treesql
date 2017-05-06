# TreeSQL

A simple database to explore alternative query languages which return trees of data instead of tables (hence the name).

Currently uses [BoltDB](https://github.com/boltdb/bolt) as a storage layer, which means it can't do concurrent write transactions.

## Example

```
localhost:9000> MANY blog_posts { id, title, body, comments: MANY comments { id, author: ONE users { name }, body } }
[
  {
    "id": "0",
    "title": "inserting stuff?",
    "body": "idk maybe it will work",
    "comments": [
      {
        "id": "0",
        "author": {
          "name": "Pete"
        },
        "body": "fa la la la la comment"
      }
    ]
  },
  {
    "id": "11",
    "title": "whooo",
    "body": "k seem to have writes",
    "comments": [
      {
        "id": "44",
        "author": {
          "name": "Pete"
        },
        "body": "whaddup"
      }
    ]
  }
]
localhost:9000>
```

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
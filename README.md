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

See `test_script.treesql` for more.

## Requirements

- Go 1.8
- [`godep`](https://github.com/tools/godep) (install with `go get github.com/tools/godep`)
- ```$ make deps```
- ```$ make```

## Usage

`cd` into this directory, then:

Start the server:

```
$ make start
```

Browse to http://localhost:9000/ and type in queries in the Web UI; e.g.:

```
many blog_posts { title, comments: many comments { body } } live
```

More examples live in `test_script.treesql`.

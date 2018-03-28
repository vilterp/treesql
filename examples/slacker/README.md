# Slacker

Tiny Slack clone to show what TreeSQL can do.

## Install

1. `npm install` in this directory.
2. Build `treesql-server` and `treesql-shell`: `make` in the TreeSQL root directory.

## Setup

Create the schema:

1. Run TreeSQL on port 9000: `treesql-server`
2. Open up the Web UI (run from `webui/`)
3. `cat setup.treesql | treesql-shell`

## Run

1. Run TreeSQL on port 9000: `treesql-server`
2. Run Node dev server: `PORT=9090 npm start`
3. Browse to `http://localhost:9090/`

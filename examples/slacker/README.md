# Slacker

Tiny Slack clone to show what TreeSQL can do.

## Installl

```npm install```

## Setup

Create the schema:

1. Run TreeSQL on port 9000: `make start` in TreeSQL repo root
2. Open up the Web UI (run from `webui/`)
3. Paste the contents of `setup.treesql` into the input box (currently no way to do this from the command line)

## Run

1. Run TreeSQL on port 9000: `make start` in TreeSQL repo root
2. Run Node dev server: `PORT=9090 npm start`
3. Browse to `http://localhost:9090/`

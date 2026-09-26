# Trail

Trail is a local Go CLI for tracking project work in Markdown and preparing daily standups.

## Developer setup

Install Go 1.26.3 or later, then from the project root:

```sh
go build -o trail ./cmd/cli
go test ./...
```

## Usage

```sh
./trail init ~/trail-data  # Set the data directory
./trail init               # Choose a location interactively
```

Run `./trail init --help` for details.

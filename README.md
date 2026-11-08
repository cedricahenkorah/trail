# Trail

Trail is a local Go CLI for tracking project work in Markdown and preparing daily standups.

## Install

With Go 1.26.3 or later:

```sh
go install github.com/cedricahenkorah/trail/cmd/trail@latest
```

Or download a prebuilt binary for macOS, Linux, or Windows from the
[Releases page](https://github.com/cedricahenkorah/trail/releases):

1. Download the archive for your platform, e.g. `trail_0.1.0_darwin_arm64.tar.gz`
   for Apple Silicon Macs.
2. Extract it and move `trail` onto your `PATH`:

   ```sh
   tar -xzf trail_0.1.0_darwin_arm64.tar.gz
   sudo mv trail /usr/local/bin/
   ```

3. Check it works with `trail --version`.

On macOS, a downloaded binary may be blocked by Gatekeeper because it isn't
signed. Clear the quarantine flag with `xattr -d com.apple.quarantine trail`.

## Developer setup

Install Go 1.26.3 or later, then from the project root:

```sh
go build -o trail ./cmd/trail
go test ./...
```

## Usage

```sh
./trail init ~/trail-data  # Set the data directory
./trail init               # Choose a location interactively
```

Run `./trail init --help` for details.

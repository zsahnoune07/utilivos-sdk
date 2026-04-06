# Install on Linux Mint

## Prerequisites

- Go 1.22+
- `make`
- optional: `dpkg-deb` (for `.deb` packaging)

Install prerequisites on Mint/Ubuntu:

```bash
sudo apt update
sudo apt install -y golang-go make dpkg-dev
```

## Local install from source

```bash
make build
sudo make install
```

Build first as your normal user. `make install` only installs the existing `bin/utilivos`; it does not rebuild the binary.

If `make build` fails with `go: command not found`, verify the toolchain:

```bash
which go
go version
```

If those commands fail, install Go first:

```bash
sudo apt update
sudo apt install -y golang-go make dpkg-dev
```

If you installed Go manually under `/usr/local/go/bin`, export it before building:

```bash
export PATH="/usr/local/go/bin:$PATH"
make build
sudo make install
```

Default install paths:

- `/usr/local/bin/utilivos`
- `/usr/local/bin/create`
- `/usr/local/bin/verify`
- `/usr/local/bin/test`
- `/usr/local/bin/run`
- `/usr/local/bin/build`

## Uninstall

```bash
sudo make uninstall
```

## Build and install .deb package

```bash
make deb
sudo dpkg -i dist/utilivos-sdk_<version>_amd64.deb
```

If dependencies are missing:

```bash
sudo apt --fix-broken install
```

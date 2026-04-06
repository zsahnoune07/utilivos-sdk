# utilivos-sdk

`utilivos-sdk` is a lightweight, strict Go CLI for building Utilivos plugins end-to-end.

The Flutter mobile shell is the single source of truth for design tokens. Plugins consume the shared web UI kit generated from that token set and rendered inside the host `WebView`.

Primary command:

```bash
utilivos
```

Primary workflow:

```bash
utilivos create my-plugin
cd my-plugin
utilivos run
utilivos verify
utilivos test
utilivos build
```

## Features

- Strict plugin scaffold generation
- Browser-ready local dev runtime (`utilivos run`) with live refresh
- Strict manifest-driven verification
- Built-in plugin tests + optional external JS test detection
- Deterministic packaging (`.ubb`)
- SHA-256 hashing
- Ed25519 detached signatures
- Archive inspection and environment diagnostics
- Linux Mint friendly install path (`make install`) and `.deb` packaging support

## Command summary

- `utilivos create <plugin-dir> [--template basic|widget] [--runtime-mode bundled|host]`
- `utilivos run [--dir path] [--host 127.0.0.1] [--port 5173|0] [--no-open] [--theme system|light|dark]`
- `utilivos verify [--json] [--strict] [--dir path]`
- `utilivos test [--json] [--filter all|builtin|external] [--dir path]`
- `utilivos pack [--out dir] [--skip-verify]`
- `utilivos sign <file-or-archive> [--key path] [--out file.sig]`
- `utilivos build [--skip-tests] [--key path] [--out dir]`
- `utilivos inspect <archive.ubb> [--json]`
- `utilivos doctor [--json]`
- `utilivos keygen [--out dir] [--name prefix]`
- `utilivos clean [--dir path]`
- `utilivos manifest lint|format|bump|update|sync` (run from plugin root, or `--dir path`; `update` and `sync` add missing manifest keys; `--dry-run` previews changes)

Full command docs: [`docs/COMMANDS.md`](docs/COMMANDS.md)
UI framework docs: [`docs/UI_FRAMEWORK.md`](docs/UI_FRAMEWORK.md)

## Getting started on Linux Mint

`utilivos-sdk` is a Go CLI. On Mint, the normal flow is: install Go and `make`, build the binary, install it into your `PATH`, verify the CLI, then scaffold your first plugin.

### 1. Install prerequisites

```bash
sudo apt update
sudo apt install -y golang-go make dpkg-dev pngquant
go version
```

`utilivos-sdk` expects Go 1.22 or newer. If `go version` reports an older release, install a newer Go toolchain first and make sure that version is the one on your `PATH`.
`pngquant` is required for manifest icon PNG compression during `utilivos pack`/`utilivos build`.

### 2. Build the CLI

From the SDK directory:

```bash
cd utilivos-sdk
make build
```

That produces:

```text
bin/utilivos
```

If you only want to try the CLI without installing it yet:

```bash
go run ./cmd/utilivos --help
```

Build first as your normal user. Do not use `sudo make build`.

### 3. Install the CLI

System-wide install to `/usr/local/bin`:

```bash
sudo make install
```

`make install` only installs `bin/utilivos`. It does not rebuild the binary, so the correct order is always:

```bash
make build
sudo make install
```

This installs:

- `/usr/local/bin/utilivos`
- `/usr/local/bin/create`
- `/usr/local/bin/verify`
- `/usr/local/bin/test`
- `/usr/local/bin/run`
- `/usr/local/bin/build`

The extra commands are wrapper aliases for the same binary.

User-local install without `sudo`:

```bash
make install PREFIX="$HOME/.local"
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Use the user-local install if you do not want to write to `/usr/local/bin`.

### Troubleshooting: `go: command not found`

If you see this during install:

```text
/bin/bash: line 1: go: command not found
```

that means the Go toolchain is not available in the shell that ran `make build`.

Check it directly:

```bash
which go
go version
```

If those commands fail, install Go first:

```bash
sudo apt update
sudo apt install -y golang-go make dpkg-dev
```

If `go` works in your user shell but fails only with `sudo make install`, your old flow was rebuilding as `root`. Use this instead:

```bash
make build
sudo make install
```

If you installed Go manually under `/usr/local/go/bin`, make sure your shell exports it before building:

```bash
export PATH="/usr/local/go/bin:$PATH"
which go
go version
make build
sudo make install
```

### 4. Verify the install

```bash
which utilivos
utilivos --help
utilivos version
```

If you installed to `~/.local/bin` and `which utilivos` returns nothing, your shell `PATH` is not updated yet.

### 5. Create and build your first plugin

```bash
mkdir -p ~/Code/utilivos
cd ~/Code/utilivos
utilivos create hello-world
cd hello-world
utilivos run
utilivos verify
utilivos test
utilivos build
```

Expected result:

- `utilivos create` scaffolds a plugin with the required manifest, localized string files, shared UI kit, and test files
- `utilivos verify` checks structure, manifest rules, design-system usage, and localization rules
- `utilivos run` starts a local browser dev shell with runtime metadata injection and live refresh
- `utilivos test` runs built-in checks and optional external JS tests
- `utilivos build` produces the signed `.ubb` release artifacts in `dist/`

### 6. Update an existing source install

If you installed from source and want the latest local version after pulling changes:

```bash
cd /path/to/utilivos-sdk
git pull
make build
sudo make install
```

If you installed to `~/.local/bin`, use:

```bash
cd /path/to/utilivos-sdk
git pull
make build
make install PREFIX="$HOME/.local"
```

### 7. Remove the SDK

Remove a system-wide source install:

```bash
cd /path/to/utilivos-sdk
sudo make uninstall
```

Remove a user-local source install:

```bash
cd /path/to/utilivos-sdk
make uninstall PREFIX="$HOME/.local"
```

That removes:

- `utilivos`
- `create`
- `verify`
- `test`
- `run`
- `build`

### 8. Optional: install from a `.deb` package

Build the package:

```bash
make deb
```

Install it:

```bash
sudo dpkg -i dist/utilivos-sdk_<version>_amd64.deb
```

If `dpkg` reports missing dependencies:

```bash
sudo apt --fix-broken install
```

Remove the `.deb` install:

```bash
sudo apt remove utilivos-sdk
```

For packaging-specific notes, see [`docs/INSTALL_LINUX_MINT.md`](docs/INSTALL_LINUX_MINT.md).

## Required plugin structure

Scaffolded by `utilivos create`:

```text
my-plugin/
├── manifest.json
├── AGENTS.md
├── build.ignore
├── README.md
├── PLAN.md
├── .env.example
├── tool/
│   ├── index.html
│   ├── main.js
│   ├── style.css
│   ├── locales/
│   │   └── en.json
│   ├── utilivos/
│   │   ├── base.css
│   │   ├── components.css
│   │   ├── layout.css
│   │   ├── utilities.css
│   │   ├── patterns.css
│   │   └── tool.js
│   └── assets/
│       ├── icon.png
│       ├── thumb.png
│       └── screenshots/home.png
├── tests/
├── dist/
└── .utilivos/project.json
```

## Manifest requirements

Required fields include:

- `id`
- `name`
- `slug`
- `version`
- `description`
- `author`
- `entry`
- `icon`
- `thumb`
- `category`
- `tags`
- `tier` (`free` | `premium` — Pro in the host app; literal `premium` in manifest)
- `permissions`
- `ui_type`
- `api_version`
- `screenshots`
- `min_app_version`
- `locales`
- `default_locale`
- `design_system_version`
- `supports_dark_mode`

Optional fields supported by strict parsing:

- `homepage`
- `repository`
- `license`
- `keywords`
- `locales`
- `default_locale`
- `content_security_policy`

Strict validation includes presence, type, enum constraints, path checks, and unresolved placeholder detection.
It also enforces the shared plugin design system imports, English-default locale resources, and prevents plugin code from overriding core `--uv-*` tokens or hardcoding user-facing copy outside locale files.

Current allowlists:

- `category`: `productivity`, `utilities`, `developer-tools`, `writing`, `design`, `automation`, `finance`, `health`, `education`, `media`, `shopping`, `communication`, `lifestyle`, `business`, `travel`, `custom`
- `tier`: `free`, `premium` (Pro tier in product copy)
- `ui_type`: `page`, `widget`, `modal`, `panel`, `background`
- `permissions`: `storage`, `clipboard.read`, `clipboard.write`, `network`, `notifications`, `camera`, `microphone`, `share`, `filesystem.read`, `filesystem.write`

## Build artifacts

`utilivos build` outputs to `dist/`:

- `<slug>-<version>.ubb`
- `<slug>-<version>.manifest.json`
- `<slug>-<version>.sha256`
- `<slug>-<version>.sig`

`utilivos pack` and `utilivos build` honor `build.ignore` when it exists.
Use one relative path per line to keep source-only files out of the final `.ubb`.
The default scaffold excludes `README.md` and `PLAN.md`.
When the manifest icon is a PNG, packaging scales it down (if needed) so the longest side is at most `512px` (aspect ratio preserved), recompresses it inside the archive, and does not change the source file on disk.

## Environment variables

- `UTILIVO_PRIVATE_KEY` - optional override for signing key path used by `utilivos build`/`utilivos sign`

## Local development

See: [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)

Quick start:

```bash
make test
make build
make run-help
```

## Debian package (.deb)

Build package:

```bash
make deb
```

Installs package file under `dist/`.

## Release process

See: [`docs/RELEASE.md`](docs/RELEASE.md)

## Example plugin

A complete example plugin is included at:

- [`examples/hello-world`](examples/hello-world)

## CI

GitHub Actions CI runs:

- `go test ./...`
- `go build ./cmd/utilivos`

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml)

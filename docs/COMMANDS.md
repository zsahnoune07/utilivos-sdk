# Command usage

## create

```bash
utilivo create <plugin-dir> [--name "My Plugin"] [--slug my-plugin] [--category utilities] [--tier free] [--author "Name"] [--description "..."] [--tags a,b,c] [--template basic|widget]
```

Creates a strict plugin scaffold with the host-aligned web UI kit, `manifest.json`, `AGENTS.md`, `PLAN.md`, `.env.example`, `build.ignore`, and metadata assets.

## run

```bash
utilivo run [--dir .] [--host 127.0.0.1] [--port 5173|0] [--no-open] [--theme system|light|dark]
```

Starts a local browser-ready dev runtime for the current plugin project.

Highlights:

- opens a desktop browser by default (disable with `--no-open`)
- serves a dev shell at `/` with the manifest entry page loaded inside it
- injects a minimal host runtime object (`window.UtilivoHost`) for parity-focused local testing
- includes live refresh on source changes via a lightweight `/__utilivo__/live` endpoint
- overlays runtime JavaScript errors in-browser for faster debugging

## verify

```bash
utilivo verify [--dir .] [--strict] [--json]
```

Validates manifest and structure strictly.

Checks include:

- required files and directories
- manifest required fields and enums
- slug/id/version validation
- `entry`, `icon`, `thumb`, `screenshots` path existence
- locale declarations and locale JSON resources
- `AGENTS.md` and `tests/` existence
- `PLAN.md` and `.env.example` guideline recommendations
- unresolved placeholders
- remote script tag restrictions
- required `utilivo/base.css`, `utilivo/layout.css`, `utilivo/components.css`, and `utilivo/tool.js` imports
- `uv-tool-page` container usage
- design system manifest fields and dark-mode support
- `data-i18n` markers and `UtilivoUI.initLocalization(...)` usage
- rejection of hardcoded user-facing strings outside `tool/locales/*.json`
- rejection of plugin-authored `--uv-*` token overrides
- package-file safety checks

`--strict` fails on warnings (CI mode).

## test

```bash
utilivo test [--dir .] [--filter all|builtin|external] [--json]
```

Runs built-in SDK tests (verification checks) and optionally external JS tests when `package.json` test script is detected.

## pack

```bash
utilivo pack [--dir .] [--out dist] [--skip-verify] [--strict] [--json]
```

Creates deterministic unsigned package artifacts.
The archive artifact extension is `.ubb`.
If `build.ignore` exists, each relative path listed there is excluded from the packaged payload.
If the manifest icon is a PNG, packaging normalizes it to `512x512` and recompresses it inside the archive.

## sign

```bash
utilivo sign <file-or-archive> [--key ~/.config/utilivo/keys/dev_ed25519_private.pem] [--out file.sig] [--json]
```

Creates detached Ed25519 signature file.

## build

```bash
utilivo build [--dir .] [--out dist] [--skip-tests] [--strict] [--key /path/private.pem] [--json]
```

Runs verify, tests (unless skipped), pack, hash, and sign.
Generated output directories are created automatically when missing.

## inspect

```bash
utilivo inspect <archive.ubb> [--json]
```

Shows archive metadata, hash, entry list, and manifest summary.

## doctor

```bash
utilivo doctor [--dir .] [--json]
```

Checks local environment health for SDK usage.

## keygen

```bash
utilivo keygen [--out ~/.config/utilivo/keys] [--name dev_ed25519] [--json]
```

Generates Ed25519 keypair for local development.

## clean

```bash
utilivo clean [--dir .]
```

Removes generated build artifacts in `dist/`.

## manifest helpers

```bash
utilivo manifest lint
utilivo manifest format
utilivo manifest bump patch|minor|major
```

## version

```bash
utilivo version
utilivo version --json
utilivo --version
```

# Command usage

## create

```bash
utilivos create <plugin-dir> [--name "My Plugin"] [--slug my-plugin] [--category utilities] [--tier free] [--author "Name"] [--description "..."] [--tags a,b,c] [--template basic|widget] [--runtime-mode bundled|host]
```

Creates a strict plugin scaffold with the host-aligned web UI kit, `manifest.json`, `AGENTS.md`, `PLAN.md`, `.env.example`, `build.ignore`, and metadata assets.

## run

```bash
utilivos run [--dir .] [--host 127.0.0.1] [--port 5173|0] [--no-open] [--theme system|light|dark]
```

Starts a local browser-ready dev runtime for the current plugin project.

Highlights:

- opens a desktop browser by default (disable with `--no-open`)
- serves a dev shell at `/` with the manifest entry page loaded inside it
- injects a minimal host runtime object (`window.UtilivosHost`) including `capabilities` (aligned with the mobile host: `restoreRequested`, `launchToken`, `slug`, etc.) for parity-focused local testing
- defines an in-memory `window.utilivos` stub (`persistState` / `loadState`) when the real bridge is absent so session sync can be exercised in the browser
- includes live refresh on source changes via a lightweight `/__utilivo__/live` endpoint
- overlays runtime JavaScript errors in-browser for faster debugging
- for `runtime_mode=host`, if `tool/utilivos/base.css` exists, injects the UI kit into the served entry HTML (your on-disk `tool/index.html` must not link `./utilivos/*`; packaging omits `tool/utilivos/` from the `.ubb` so the real app supplies the runtime)

## verify

```bash
utilivos verify [--dir .] [--strict] [--json]
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
- required `utilivos/base.css`, `utilivos/layout.css`, `utilivos/components.css`, and `utilivos/tool.js` imports
- `uv-tool-page` container usage
- design system manifest fields and dark-mode support
- `data-i18n` markers and `UtilivosUI.initLocalization(...)` usage
- rejection of hardcoded user-facing strings outside `tool/locales/*.json`
- rejection of plugin-authored `--uv-*` token overrides
- package-file safety checks
- **warning** `session.persist_bridge` when no `UtilivosRuntime.syncState` or `utilivos.persistState` usage is found under `tool/*.js` (excluding `tool/utilivos/`)

`--strict` fails on warnings (CI mode).

### Session save/load (host)

Bundled plugins ship `UtilivosRuntime` in `tool/utilivos/tool.js`. Use `UtilivosRuntime.syncState({ read, write, debounceMs })` and hydrate only when `UtilivosHost.capabilities.restoreRequested` is true (the helper enforces this). The app’s **Save session** action reads ephemeral JSON written via `utilivos.persistState`.

For the full contract (Pro gating, no auto-restore on open, payload shape), see the Utilivos repository’s [`tool_lib_updates.md`](../../tool_lib_updates.md) at the monorepo root (path is valid when this SDK lives inside the Utilivos tree).

Plugins with `runtime_mode=host` do not package `tool/utilivos/`; the Flutter host must inject a compatible UI/runtime bundle that includes the same `UtilivosRuntime` API as this SDK’s `tool.js`.

## test

```bash
utilivos test [--dir .] [--filter all|builtin|external] [--json]
```

Runs built-in SDK tests (verification checks) and optionally external JS tests when `package.json` test script is detected.

## pack

```bash
utilivos pack [--dir .] [--out dist] [--skip-verify] [--strict] [--json]
```

Creates deterministic unsigned package artifacts.
The archive artifact extension is `.ubb`.
If `build.ignore` exists, each relative path listed there is excluded from the packaged payload.
If the manifest icon is a PNG, packaging scales it so the longest side is at most `512px` (aspect ratio preserved), then recompresses it inside the archive.

## sign

```bash
utilivos sign <file-or-archive> [--key ~/.config/utilivos/keys/dev_ed25519_private.pem] [--out file.sig] [--json]
```

Creates detached Ed25519 signature file.

## build

```bash
utilivos build [--dir .] [--out dist] [--skip-tests] [--strict] [--key /path/private.pem] [--json]
```

Runs verify, tests (unless skipped), pack, hash, and sign.
Generated output directories are created automatically when missing.

## inspect

```bash
utilivos inspect <archive.ubb> [--json]
```

Shows archive metadata, hash, entry list, and manifest summary.

## doctor

```bash
utilivos doctor [--dir .] [--json]
```

Checks local environment health for SDK usage.

## keygen

```bash
utilivos keygen [--out ~/.config/utilivos/keys] [--name dev_ed25519] [--json]
```

Generates Ed25519 keypair for local development.

## clean

```bash
utilivos clean [--dir .]
```

Removes generated build artifacts in `dist/`.

## manifest helpers

Run these from the plugin root (the directory that contains `manifest.json`), or pass `--dir` to that folder.

```bash
utilivos manifest lint [--dir .]
utilivos manifest format [--dir .]
utilivos manifest bump patch|minor|major [--dir .]
utilivos manifest update [--dir .] [--dry-run]
utilivos manifest sync [--dir .] [--dry-run]
```

`update` and `sync` are the same command: they merge in any keys required by the current SDK manifest contract when those fields are missing (for example `design_system_version`, `runtime_mode`, `locales`). Use `--dry-run` to list fields that would change without writing the file.

## version

```bash
utilivos version
utilivos version --json
utilivos --version
```

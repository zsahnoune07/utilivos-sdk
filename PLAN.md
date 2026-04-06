# utilivos-sdk PLAN

## Repository goal
Build a strict, lightweight Go CLI (`utilivos`) for the Utilivos plugin ecosystem with a complete local workflow:

- `utilivos create`
- `utilivos verify`
- `utilivos test`
- `utilivos build`

Plus support commands for day-to-day development and release automation.

## Scope for v0.1.0

### Core commands
- `create` (strict scaffold generator)
- `verify` (manifest + structure + packaging readiness validation)
- `test` (built-in checks + optional external JS runner detection)
- `pack` (deterministic unsigned archive)
- `sign` (Ed25519 detached signature)
- `build` (verify + test + pack + sign)

### Additional commands
- `doctor`
- `inspect`
- `keygen`
- `clean`
- `manifest lint|format|bump`

### Plugin rules enforced
- strict required files and directories
- strict `manifest.json` parsing and field validation
- strict path/reference checks (`entry`, `icon`, `thumb`, `screenshots`)
- tier enforcement (`free` | `premium`; Pro in host UI)
- category/ui/permission allowlists
- unresolved placeholder detection
- forbidden package file detection
- deterministic `.ubb` packaging flow

### Build outputs
Generated in `dist/`:
- `<slug>-<version>.ubb`
- `<slug>-<version>.manifest.json`
- `<slug>-<version>.sha256`
- `<slug>-<version>.sig`

### Security model
- Ed25519 key generation via `utilivos keygen`
- detached signatures over archive SHA-256 digest
- dev key default path under `~/.config/utilivos/keys`
- production signing via injected key path (`--key` / env)

## Delivery checklist

1. [x] Initialize Go module and CLI architecture
2. [x] Implement `create`
3. [x] Implement `verify`
4. [x] Implement `test`
5. [x] Implement `pack` + `sign` + `build`
6. [x] Implement `doctor` + `inspect` + `keygen` + `clean`
7. [x] Add manifest helper commands
8. [x] Add fixtures and automated tests
9. [x] Add docs and examples
10. [x] Add Linux Mint install flow (`make install`) and `.deb` packaging support
11. [x] Add CI workflow

## Notes
- Keep dependencies minimal (stdlib-first).
- Keep CLI behavior strict and CI-friendly.
- Keep packaging deterministic where practical.
- Keep templates embedded via code for maintainability.

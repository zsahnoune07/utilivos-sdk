# Development

## Setup

```bash
go version
make test
make build
```

Optional local env values are documented in `.env.example`.

## Useful targets

- `make test` - run Go tests
- `make build` - build `bin/utilivo`
- `make run-help` - run CLI help via `go run`
- `make lint` - run `go vet`
- `make clean-bin` - remove built binary

## Local plugin runtime loop

From any scaffolded plugin directory:

```bash
utilivo run
```

This starts a browser dev shell with live refresh and dev runtime metadata injection.

## UI asset sync

To keep shipped `ui/*` and embedded scaffold assets aligned:

```bash
scripts/sync_ui_assets.sh
scripts/sync_ui_assets.sh --check
```

## Test fixtures

Fixtures are in `tests/fixtures/` and cover valid and invalid plugin cases for verification behavior.

## Architecture

- `cmd/utilivo` - entrypoint
- `internal/cli` - command dispatch and UX
- `internal/scaffold` - plugin generation
- `internal/manifest` - strict manifest parse/validate
- `internal/verify` - structure/readiness validation
- `internal/testing` - built-in + external tests
- `internal/archive` - deterministic packaging + inspection
- `internal/signing` - Ed25519 keygen/sign/verify
- `internal/build` - pack/sign orchestration
- `internal/doctor` - environment checks
- `internal/templates` - scaffold templates/assets
- `internal/util` - shared helpers

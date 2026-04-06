# Release process

## 1. Validate

```bash
make test
make build
```

## 2. Bump version

Update `internal/util/version.go` and if needed run:

```bash
utilivos manifest bump patch
```

## 3. Build release assets

```bash
make release
make deb
```

Expected outputs in `dist/`:

- `utilivos` binary
- `utilivos-sdk_<version>_amd64.deb`

## 4. Smoke test

```bash
./bin/utilivos --help
./bin/utilivos keygen --out /tmp/utilivos-keys
```

## 5. Publish

- tag release in git
- upload artifacts
- include checksums/signatures if required by distribution process

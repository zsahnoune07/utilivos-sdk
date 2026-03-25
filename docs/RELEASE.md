# Release process

## 1. Validate

```bash
make test
make build
```

## 2. Bump version

Update `internal/util/version.go` and if needed run:

```bash
utilivo manifest bump patch
```

## 3. Build release assets

```bash
make release
make deb
```

Expected outputs in `dist/`:

- `utilivo` binary
- `utilivo-sdk_<version>_amd64.deb`

## 4. Smoke test

```bash
./bin/utilivo --help
./bin/utilivo keygen --out /tmp/utilivo-keys
```

## 5. Publish

- tag release in git
- upload artifacts
- include checksums/signatures if required by distribution process

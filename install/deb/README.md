# Debian packaging

Use the root Makefile target:

```bash
make deb
```

This invokes `scripts/package_deb.sh`, staging the package under `.pkg/` and producing:

- `dist/utilivo-sdk_<version>_amd64.deb`

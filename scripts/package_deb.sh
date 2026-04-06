#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.1.0}"
ARCH="${ARCH:-amd64}"
PKG_NAME="utilivos-sdk"
OUT_DIR="${ROOT_DIR}/dist"
STAGE_DIR="${ROOT_DIR}/.pkg/${PKG_NAME}"

if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to build .deb packages" >&2
  exit 1
fi

if [[ ! -f "${ROOT_DIR}/bin/utilivos" ]]; then
  echo "missing bin/utilivos, run 'make build' first" >&2
  exit 1
fi

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/DEBIAN" "${STAGE_DIR}/usr/bin" "${OUT_DIR}"

cp "${ROOT_DIR}/bin/utilivos" "${STAGE_DIR}/usr/bin/utilivos"
ln -sf utilivos "${STAGE_DIR}/usr/bin/create"
ln -sf utilivos "${STAGE_DIR}/usr/bin/verify"
ln -sf utilivos "${STAGE_DIR}/usr/bin/test"
ln -sf utilivos "${STAGE_DIR}/usr/bin/run"
ln -sf utilivos "${STAGE_DIR}/usr/bin/build"

cat > "${STAGE_DIR}/DEBIAN/control" <<CONTROL
Package: ${PKG_NAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: Utilivos <dev@utilivos.local>
Depends: libc6
Description: Utilivos SDK CLI for plugin create/run/verify/test/build workflows
CONTROL

OUTPUT_DEB="${OUT_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.deb"
dpkg-deb --build "${STAGE_DIR}" "${OUTPUT_DEB}"

echo "built: ${OUTPUT_DEB}"

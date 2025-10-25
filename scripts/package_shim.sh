#!/usr/bin/env bash

set -euo pipefail

log() {
	printf '[package_shim] %s\n' "$*" >&2
}

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHIM_DIR="${PROJECT_ROOT}/swift/FundamentShim"
BUILD_ROOT="${SHIM_DIR}/.build"
RELEASE_DIR="${BUILD_ROOT}/Release"
DYLIB_NAME="libFundamentShim.dylib"
SOURCE_PATH="${RELEASE_DIR}/${DYLIB_NAME}"
TARGET_DIR="${PROJECT_ROOT}/internal/shimloader/prebuilt"
TARGET_PATH="${TARGET_DIR}/${DYLIB_NAME}"
MANIFEST_PATH="${TARGET_DIR}/manifest.json"

if [[ ! -f "${SOURCE_PATH}" ]]; then
	log "error: missing release artefact at ${SOURCE_PATH}"
	log "hint: run 'make swift' (or 'swift build -c release') before packaging"
	exit 1
fi

mkdir -p "${TARGET_DIR}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

tmp_dylib="${tmpdir}/${DYLIB_NAME}"
cp "${SOURCE_PATH}" "${tmp_dylib}"

if command -v lipo >/dev/null 2>&1; then
	if ! lipo -info "${tmp_dylib}" >/dev/null 2>&1; then
		log "warning: failed to inspect architectures with lipo"
	fi
else
	log "warning: 'lipo' not found; skipping architecture inspection"
fi

if command -v codesign >/dev/null 2>&1; then
	if ! codesign -dv "${tmp_dylib}" >/dev/null 2>&1; then
		log "warning: codesign inspection failed (dylib may be unsigned)"
	fi
else
	log "warning: 'codesign' not found; skipping signature inspection"
fi

install_name="@rpath/${DYLIB_NAME}"
if command -v install_name_tool >/dev/null 2>&1; then
	install_name_tool -id "${install_name}" "${tmp_dylib}"
else
	log "warning: 'install_name_tool' not found; keeping existing install name"
fi

new_sha="$(shasum -a 256 "${tmp_dylib}" | awk '{print $1}')"

existing_sha=""
if [[ -f "${MANIFEST_PATH}" ]]; then
	existing_sha="$(
python3 - "${MANIFEST_PATH}" <<'PY' 2>/dev/null || true
import json, sys
try:
    with open(sys.argv[1], "r", encoding="utf-8") as fh:
        data = json.load(fh)
    print(data.get("sha256", ""))
except Exception:
    print("")
PY
	)"
fi

if [[ -f "${TARGET_PATH}" ]] && cmp -s "${tmp_dylib}" "${TARGET_PATH}" && [[ -n "${existing_sha}" ]] && [[ "${existing_sha}" == "${new_sha}" ]]; then
	log "prebuilt shim already up to date (sha256=${new_sha})"
	exit 0
fi

install -m 0644 "${tmp_dylib}" "${TARGET_PATH}"

generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
swift_version="$(swiftc --version 2>/dev/null | head -n 1 | tr -d '\r')"
sdk_version="$(xcrun --sdk macosx --show-sdk-version 2>/dev/null || echo "unknown")"

cat >"${MANIFEST_PATH}" <<EOF
{
  "sha256": "${new_sha}",
  "generated_at": "${generated_at}",
  "swift_version": "${swift_version}",
  "sdk_version": "${sdk_version}"
}
EOF

log "updated prebuilt shim -> ${TARGET_PATH} (sha256=${new_sha})"

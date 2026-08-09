#!/usr/bin/env bash
# Prints -ldflags that stamp internal/version.Version from the nearest git tag
# (v1.2.3 → 1.2.3). Empty output if no usable tag.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

tag="${VERSION_TAG:-}"
if [[ -z "$tag" ]]; then
  tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
fi
if [[ -z "$tag" ]]; then
  exit 0
fi
ver="${tag#v}"
if [[ -z "$ver" ]]; then
  exit 0
fi
printf '%s' "-X github.com/migzeetigglez/ashen-sync/internal/version.Version=${ver}"

#!/usr/bin/env bash
set -euo pipefail

if [[ " $* " == *"--print-map"* ]]; then
  "${KTAV_REAL_ZIG:?}" "$@" | tee -a "${KTAV_LINK_MAP:?}"
else
  exec "${KTAV_REAL_ZIG:?}" "$@"
fi

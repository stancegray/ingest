#!/usr/bin/env bash
# Base packages needed before docker deploy (git, curl, make, etc.)
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]] && ! command -v sudo >/dev/null 2>&1; then
	echo "install-system.sh requires root or sudo" >&2
	exit 1
fi

run() {
	if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
		"$@"
	else
		sudo "$@"
	fi
}

if ! command -v apt-get >/dev/null 2>&1; then
	echo "install-system.sh supports Ubuntu/Debian (apt) only" >&2
	exit 1
fi

echo "==> Installing system packages"
run apt-get update
run apt-get install -y git curl make ca-certificates

echo "System packages installed."

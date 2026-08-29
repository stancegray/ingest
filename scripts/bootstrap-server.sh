#!/usr/bin/env bash
# Full server bootstrap: clone/pull repo, install Docker, deploy stack.
# Usage:
#   GITHUB_TOKEN=ghp_... make bootstrap
# Fresh server one-liner:
#   GITHUB_TOKEN=ghp_... bash -c "$(curl -fsSL https://raw.githubusercontent.com/stancegray/ingest/main/scripts/bootstrap-server.sh)"
set -euo pipefail

REPO_SLUG="${REPO_SLUG:-stancegray/ingest}"
REPO_BRANCH="${REPO_BRANCH:-main}"
REPO_DIR="${REPO_DIR:-ingest}"
REPO_HTTPS="https://github.com/${REPO_SLUG}.git"
AUTH_REMOTE="https://x-access-token:${GITHUB_TOKEN:?Set GITHUB_TOKEN=ghp_...}@github.com/${REPO_SLUG}.git"

if [[ -f .env ]]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

: "${GITHUB_TOKEN:?Set GITHUB_TOKEN=ghp_... (or add to .env)}"

ensure_git() {
	if command -v git >/dev/null 2>&1; then
		return
	fi
	if command -v apt-get >/dev/null 2>&1; then
		echo "==> Installing git (required to clone repo)"
		sudo apt-get update
		sudo apt-get install -y git
		return
	fi
	echo "git is required" >&2
	exit 1
}

ensure_git

if [[ ! -d .git ]]; then
	if [[ -d "${REPO_DIR}/.git" ]]; then
		cd "${REPO_DIR}"
	else
		echo "==> Cloning ${REPO_HTTPS}"
		git clone --branch "${REPO_BRANCH}" "${AUTH_REMOTE}" "${REPO_DIR}"
		cd "${REPO_DIR}"
	fi
fi

echo "==> Pulling latest ${REPO_BRANCH}"
git remote set-url origin "${REPO_HTTPS}"
git fetch "${AUTH_REMOTE}" "${REPO_BRANCH}"
git checkout "${REPO_BRANCH}"
git reset --hard "FETCH_HEAD"

echo "==> Installing system packages"
make install

echo "==> Installing Docker (official Ubuntu guide)"
make install-deps

echo "==> Deploying postgres + ingest"
if groups | grep -qw docker; then
	make deploy
else
	echo "Docker group not active in this shell; using sudo for deploy."
	sudo -E make deploy
fi

echo ""
echo "Bootstrap complete."

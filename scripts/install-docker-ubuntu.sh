#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]] && ! command -v sudo >/dev/null 2>&1; then
	echo "install-docker-ubuntu.sh requires root or sudo" >&2
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
	echo "This installer supports Ubuntu/Debian (apt) only." >&2
	echo "See https://docs.docker.com/engine/install/ubuntu/" >&2
	exit 1
fi

if [[ -r /etc/os-release ]]; then
	# shellcheck source=/dev/null
	. /etc/os-release
	if [[ "${ID:-}" != "ubuntu" && "${ID_LIKE:-}" != *debian* ]]; then
		echo "Warning: official guide targets Ubuntu; continuing on ${PRETTY_NAME:-unknown}"
	fi
fi

echo "==> Removing conflicting packages (if present)"
conflicts=(docker.io docker-compose docker-compose-v2 docker-doc docker-buildx podman-docker containerd runc)
installed=()
for pkg in "${conflicts[@]}"; do
	if dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
		installed+=("$pkg")
	fi
done
if ((${#installed[@]} > 0)); then
	run apt-get remove -y "${installed[@]}"
else
	echo "    none installed"
fi

echo "==> Installing prerequisites"
run apt-get update
run apt-get install -y ca-certificates curl

echo "==> Adding Docker apt repository"
run install -m 0755 -d /etc/apt/keyrings
run curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
run chmod a+r /etc/apt/keyrings/docker.asc

codename="$(. /etc/os-release && echo "${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}")"
arch="$(dpkg --print-architecture)"
if [[ -z "$codename" ]]; then
	echo "Could not detect Ubuntu codename from /etc/os-release" >&2
	exit 1
fi

run tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: ${codename}
Components: stable
Architectures: ${arch}
Signed-By: /etc/apt/keyrings/docker.asc
EOF

echo "==> Installing Docker Engine"
run apt-get update
run apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

echo "==> Enabling Docker service"
run systemctl enable --now docker

echo "==> Verifying installation"
run docker run --rm hello-world >/dev/null
run docker compose version

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	if id -nG "$USER" | grep -qw docker; then
		echo "User $USER is already in the docker group."
	else
		echo "==> Adding $USER to docker group (log out/in or run: newgrp docker)"
		run usermod -aG docker "$USER"
	fi
fi

echo ""
echo "Docker Engine installed successfully."
echo "Docs: https://docs.docker.com/engine/install/ubuntu/"

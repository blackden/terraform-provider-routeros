#!/usr/bin/env bash
# Build the blackden fork and (optionally) install it into an OpenTofu filesystem mirror.
#
#   fork/build.sh            # test + build dist/<os>_<arch>/terraform-provider-routeros_v<version>
#   fork/build.sh install    # ... and copy into $FORK_MIRROR (default ~/.terraform.d/plugins)
#   SKIP_TESTS=1 fork/build.sh
#   FORK_PLATFORMS="linux_amd64" fork/build.sh   # limit target platforms
#   FORK_SOURCE=example.org/ns/routeros fork/build.sh install   # mirror source address
#
# See FORK.md.
set -euo pipefail

cd "$(dirname "$0")/.."

version=$(cat FORK_VERSION)
source_addr=${FORK_SOURCE:-registry.opentofu.org/blackden/routeros}
mirror=${FORK_MIRROR:-$HOME/.terraform.d/plugins}
platforms=${FORK_PLATFORMS:-"linux_amd64 darwin_arm64 darwin_amd64 linux_arm64"}

if [[ -z "${SKIP_TESTS:-}" ]]; then
	go build ./...
	go vet ./routeros ./tools/... || true # upstream has a known vet warning (resource_interface_wireless_test.go)
	for v in 7.23 7.24; do
		ROS_VERSION=$v go test ./routeros -skip TestClientTransport_SendRequest
	done
fi

for p in $platforms; do
	os=${p%_*} arch=${p#*_}
	echo "build $p v$version"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" \
		-o "dist/$p/terraform-provider-routeros_v$version" .
done
(cd dist && sha256sum */"terraform-provider-routeros_v$version" > "SHA256SUMS_v$version")
cat "dist/SHA256SUMS_v$version"

if [[ "${1:-}" == install ]]; then
	for p in $platforms; do
		d="$mirror/$source_addr/$version/$p"
		mkdir -p "$d"
		install -m 0755 "dist/$p/terraform-provider-routeros_v$version" "$d/"
		echo "installed $d"
	done
fi

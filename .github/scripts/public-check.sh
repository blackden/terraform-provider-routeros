#!/usr/bin/env bash
# Usage: public-check.sh <base-rev>
# Checks files changed between <base-rev> and HEAD: no Cyrillic text anywhere,
# and no non-documentation IPv4 addresses in examples/ and docs/.
# Exit 1 and print "public-check: <file>:<line>" for every finding.
set -u
export LC_ALL=C.UTF-8

base=${1:?usage: public-check.sh <base-rev>}
rc=0

# Fail closed: a missing base or any git failure must not look like "no findings".
git rev-parse --verify -q "$base^{commit}" >/dev/null || { echo "public-check: base $base not found" >&2; exit 2; }
files=$(git -c core.quotePath=false diff --name-only --diff-filter=ACMR "$base" HEAD) || { echo "public-check: git diff failed" >&2; exit 2; }

# Allowed: 192.0.2/24, 198.51.100/24, 203.0.113/24, 127/8, 0.0.0.0, 10/8, 172.16/12, 192.168/16
octet='(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])'
ipv4="(?<![0-9.])${octet}(\\.${octet}){3}(?![0-9])"
allowed='^(192\.0\.2\.|198\.51\.100\.|203\.0\.113\.|127\.|0\.0\.0\.0$|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.)'
# Netmasks (255.255.255.0, 128.0.0.0, ...) are not addresses either
m='(128|192|224|240|248|252|254)'
masks="^((${m}|255)\\.0\\.0\\.0|255\\.(${m}|255)\\.0\\.0|255\\.255\\.(${m}|255)\\.0|255\\.255\\.255\\.(0|${m}|255))\$"

while IFS= read -r f; do
	[ -n "$f" ] || continue
	if printf '%s' "$f" | grep -qP '[\x{0400}-\x{04FF}]'; then
		echo "public-check: $f (cyrillic in file name)"
		rc=1
	fi
	[ -f "$f" ] || continue
	grep -Iq . "$f" 2>/dev/null || continue
	while IFS=: read -r n _; do
		echo "public-check: $f:$n (cyrillic)"
		rc=1
	done < <(grep -nP '[\x{0400}-\x{04FF}]' "$f")
	case "$f" in
	examples/* | docs/*)
		while IFS=: read -r n ip; do
			if ! printf '%s\n' "$ip" | grep -qE "$allowed|$masks"; then
				echo "public-check: $f:$n (ipv4 $ip)"
				rc=1
			fi
		done < <(grep -noP "$ipv4" "$f")
		;;
	esac
done <<<"$files"

exit $rc

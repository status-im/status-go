#!/usr/bin/env bash
# Runs the //go:generate directives the library build needs with their output
# redirected under <out-dir>, and writes <out-dir>/overlay.json for
# `go build -overlay`. Nothing is written into the source tree, so a read-only
# copy of it (nimble's package store) builds without tracked generated files.
#
# The directives stay the single source of truth: each one is run from its own
# directory with only its output path rewritten. A directive whose shape is not
# known here is an error, not a skip. Mocks and contract bindings are test- or
# developer-only and are left to `make generate`.
set -euo pipefail

[[ $# -eq 1 ]] || { echo "usage: $0 <out-dir>" >&2; exit 2; }
# Git Bash on Windows: `pwd -W` is the native form (J:/...). The Go toolchain
# reads overlay.json, and it does not understand /j/... paths.
native_pwd() { pwd -W 2>/dev/null || pwd; }
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && native_pwd)
mkdir -p "$1"
out=$(cd "$1" && native_pwd)
gen="$out/src"
rm -rf "$gen"

run_directive() {
	local file=$1 cmd=$2 dir rel
	dir=$(dirname "$file")
	rel=${dir#"$root"/}
	mkdir -p "$gen/$rel"
	case $cmd in
	'protoc "--go_prefix=go tool" --go_out=. '*)
		(cd "$dir" && eval "${cmd/ --go_out=. / --go_out=\"$gen/$rel\" }")
		;;
	'go tool go-bindata '*' -o ../'*)
		(cd "$dir" && eval "${cmd/ -o ..\// -o \"$gen/$rel\"/../}")
		;;
	'go run main.go')
		mkdir -p "$gen/cmd/status-backend/server" "$gen/internal/protocol"
		(cd "$dir" && STATUSGO_GENERATE_ROOT="$gen" go run main.go)
		;;
	*)
		echo "generate-overlay: unsupported directive in ${file#"$root"/}: $cmd" >&2
		exit 1
		;;
	esac
}

# Searched from the root so a hit reads ./path:directive: a native Windows root
# (J:/...) would put a second colon in front of the one that ends the path.
while IFS= read -r hit; do
	file=${hit%%:*}
	cmd=${hit#*://go:generate }
	run_directive "$root/${file#./}" "$cmd"
done < <(cd "$root" && grep -r --include='*.go' --exclude-dir=vendor '^//go:generate ' . |
	grep -v -e 'mockgen' -e 'abigen' -e '/contracts/' | sort)

# {"Replace": {"<tree>/<path>.go": "<out-dir>/src/<path>.go"}}
{
	printf '{"Replace":{'
	sep=
	while IFS= read -r f; do
		printf '%s\n"%s":"%s"' "$sep" "$root/${f#"$gen"/}" "$f"
		sep=,
	done < <(find "$gen" -name '*.go' | sort)
	printf '\n}}\n'
} > "$out/overlay.json"
echo "generate-overlay: $(grep -c '":"' "$out/overlay.json") files under $gen"

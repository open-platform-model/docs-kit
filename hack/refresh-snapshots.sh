#!/usr/bin/env bash
# Refresh one release snapshot under internal/render/testdata/<kind>/ from a
# released docs bundle: verify and unpack the bundle with this checkout's
# opm-docs pull, copy its pages under the owned section to want/, take the
# inputs the pages were built from out of the release commit, and write
# SOURCE. TestControllerCRDParity (kind controller) and TestCLICommandParity
# (kind cli) compare what this checkout renders from the inputs with want/.
#
#   hack/refresh-snapshots.sh controller <project> <version> <checkout>
#   hack/refresh-snapshots.sh cli        <project> <version> <checkout>
#
# <project> is the docs bundle's project (opm-controller, cli), signed by
# open-platform-model/<project>; <version> is the release without "v"
# (1.0.0-beta.9), whose newest docs revision is pulled; <checkout> is a clone
# of that repository holding the release commit. The cli kind runs the
# release's hack/docskit-dump twice in an exported copy of the release tree
# and refuses differing output (C14). Needs network (GHCR, the Go proxy),
# git, jq and tar.
set -euo pipefail

usage() {
	echo "usage: $0 <controller|cli> <project> <version> <checkout>" >&2
	exit 2
}
[ $# -eq 4 ] || usage
kind=$1 project=$2 version=$3 checkout=$4
case $kind in controller | cli) ;; *) usage ;; esac
case $version in v*) echo "$0: give the version without v: ${version#v}" >&2; exit 2 ;; esac

root=$(cd "$(dirname "$0")/.." && pwd)
snap=$root/internal/render/testdata/$kind
tmp=$(mktemp -d)
trap 'chmod -R u+w "$tmp" 2>/dev/null; rm -rf "$tmp"' EXIT

# Pull the bundle as the site would: signature, signer identity and the
# manifest verified before anything unpacks.
major_minor=$(echo "$version" | cut -d. -f1-2)
site=v$major_minor
cat >"$tmp/bundles.cue" <<EOF
docs: "$project": repo: "open-platform-model/$project"
versions: "$site": anchor: {project: "$project", tag: "$version"}
EOF
(cd "$root" && go run ./cmd/opm-docs pull --config "$tmp/bundles.cue" --out "$tmp/out")

lock=$tmp/out/lock.json
bundle=$tmp/out/$(jq -r '.docs[0].dir' "$lock")
commit=$(jq -r '.docs[0].commit' "$lock")
section=$(jq -r '.placement.owns | if length == 1 then .[0] else error("the bundle owns \(length) paths, want one") end' "$bundle/manifest.json")
case $section in */) ;; *) echo "$0: owned path $section is not a section" >&2; exit 1 ;; esac

# The snapshot's inputs, taken from the release commit.
inputs=(docs-kit.cue)
if [ "$kind" = controller ]; then
	inputs+=(config/crd/bases config/samples)
	if git -C "$checkout" cat-file -e "$commit:docs/site/${section}_index.md" 2>/dev/null; then
		inputs+=("docs/site/${section}_index.md")
	fi
fi

# A docs revision builds the release commit with fix commits applied; the
# snapshot reads the release commit, so no fix may touch an input.
for patch in $(jq -r '.source.patches // [] | .[]' "$bundle/manifest.json"); do
	for f in $(git -C "$checkout" show --name-only --format= "$patch"); do
		for in in "${inputs[@]}"; do
			case $f in "$in" | "$in"/*)
				echo "$0: docs revision fix $patch changes $f, an input of the snapshot" >&2
				exit 1
				;;
			esac
		done
	done
done

new=$tmp/snap
mkdir -p "$new/want"
git -C "$checkout" archive "$commit" -- "${inputs[@]}" | tar -x -C "$new"
if [ "$kind" = cli ]; then
	mkdir "$tmp/src"
	git -C "$checkout" archive "$commit" | tar -x -C "$tmp/src"
	(cd "$tmp/src" && go run ./hack/docskit-dump >"$tmp/dump1.json" && go run ./hack/docskit-dump >"$tmp/dump2.json")
	cmp -s "$tmp/dump1.json" "$tmp/dump2.json" || { echo "$0: two runs of hack/docskit-dump differ" >&2; exit 1; }
	cp "$tmp/dump1.json" "$new/dump.json"
fi
mkdir -p "$new/want/$section"
cp -R "$bundle/content/$section." "$new/want/$section"

jq -r --arg kind "$kind" --arg section "$section" '.docs[0] |
"Release snapshot for docs-kit'"'"'s parity tests (contracts C18, C19).

Bundle:   \(.repository):\(.version) (docs revision \(.revision))
Digest:   \(.digest)
Built by: opm-docs \(.builtBy)
Source:   open-platform-model/\(.project) at \(.commit) (Apache-2.0)

want/ holds the bundle'"'"'s pages under \($section). " +
(if $kind == "controller" then "docs-kit.cue, config/crd/bases,
config/samples and the authored section index under docs/site/ are the
release commit'"'"'s inputs to them."
else "docs-kit.cue is the release
commit'"'"'s; dump.json is what its hack/docskit-dump printed, twice alike." end) + "

Refresh with task snapshots:refresh (hack/refresh-snapshots.sh), never by
hand."' "$lock" >"$new/SOURCE"

rm -rf "$snap"
mv "$new" "$snap"
echo "refreshed $snap from $project $version ($(jq -r '.docs[0].digest' "$lock"))"

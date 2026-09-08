#!/bin/bash
# Check the trusted in-repository recipe without sourcing a PKGBUILD.
set -euo pipefail
root=$(git -C "$(dirname -- "$0")" rev-parse --show-toplevel)
cd "$root"
recipe=$(<packaging/aur/PKGBUILD)
commit_pattern="^_commit='([0-9a-f]{40})'$"
version_pattern='^pkgver=([0-9]+\.[0-9]+\.[0-9]+\.r[0-9]+\.g([0-9a-f]{7}))$'
commit=''
version=''
short=''
while IFS= read -r line; do
  if [[ $line =~ $commit_pattern ]]; then commit=${BASH_REMATCH[1]}; fi
  if [[ $line =~ $version_pattern ]]; then
    version=${BASH_REMATCH[1]}
    short=${BASH_REMATCH[2]}
  fi
done <<<"$recipe"
if [[ -z $commit || -z $version || ${commit:0:7} != "$short" ]]; then
  echo 'Package freshness: invalid or inconsistent _commit/pkgver.' >&2
  exit 1
fi
git merge-base --is-ancestor "$commit" HEAD || {
  echo 'Package freshness: source pin must be preserved in branch history.' >&2
  exit 1
}
# These are the source, test, dependency and documentation inputs shipped by
# the current recipe. Extend this list when build()/check()/package() changes.
inputs=(cmd internal go.mod go.sum README.md)
if ! git diff --quiet "$commit" -- "${inputs[@]}" ||
   [[ -n $(git ls-files --others --exclude-standard -- "${inputs[@]}") ]]; then
  echo 'Package freshness: source pin is stale; update _commit, pkgver, checksum and .SRCINFO.' >&2
  git diff --stat "$commit" -- "${inputs[@]}"
  exit 1
fi
printf 'Package freshness OK: %s (%s)\n' "$version" "$commit"

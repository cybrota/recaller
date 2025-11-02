#!/usr/bin/env bash
set -euo pipefail

# Guard rails to ensure CLI version output matches release tag.

if [[ ${GORELEASER_SNAPSHOT:-} == "true" ]]; then
  exit 0
fi

version_file="version.go"
if [[ ! -f "$version_file" ]]; then
  echo "verify.sh: missing ${version_file}" >&2
  exit 1
fi

if ! version_line=$(grep -E 'const[[:space:]]+version[[:space:]]*=' "$version_file"); then
  echo "verify.sh: could not find version constant in ${version_file}" >&2
  exit 1
fi

if [[ ! $version_line =~ \"([^\"]+)\" ]]; then
  echo "verify.sh: failed to parse version constant in ${version_file}" >&2
  exit 1
fi

file_version="${BASH_REMATCH[1]}"
tag_version="${GORELEASER_CURRENT_TAG:-}"

if [[ -z "$tag_version" ]]; then
  tag_version=$(git describe --tags --exact-match 2>/dev/null || true)
fi

if [[ -z "$tag_version" ]]; then
  echo "verify.sh: no release tag detected; aborting" >&2
  exit 1
fi

if [[ "$file_version" != "$tag_version" ]]; then
  cat <<EOF >&2
verify.sh: version mismatch
  version.go: ${file_version}
  release tag: ${tag_version}
EOF
  exit 1
fi

echo "verify.sh: version ${file_version} matches release tag ${tag_version}"

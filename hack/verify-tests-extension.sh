#!/usr/bin/env bash

set -euo pipefail

project_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
binary=${1:?extension binary path is required}
metadata_dir="$project_dir/hack/.openshift-tests-extension"
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

# Preserve previous metadata so OTE can also validate removed/renamed tests.
cp -R "$metadata_dir" "$work_dir/.openshift-tests-extension"
(cd "$work_dir" && "$binary" update)
diff -ru "$metadata_dir" "$work_dir/.openshift-tests-extension"

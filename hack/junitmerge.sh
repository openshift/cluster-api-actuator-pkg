#!/bin/bash

set -e

function find::junitmerge () {
  local binary_name="junitmerge"

	local old_ifs="${IFS}"
	IFS=":"
	for part in ${GOPATH}; do
		local binary_path="${part}/bin/${binary_name}"
		# we need to check that the path leads to a file
		# as directories also have the executable bit set
		if [[ -f "${binary_path}" && -x "${binary_path}" ]]; then
			echo "${binary_path}"
			IFS="${old_ifs}"
			return 0
		fi
	done
	IFS="${old_ifs}"
	return 1
}

function ensure::junitmerge () {
  if ! find::junitmerge >/dev/null 2>&1; then
    # File does not exist, fetch it
    GO111MODULE=off go get github.com/openshift/release/tools/junitmerge
  fi
}

# Check there are files to merge
if ! ls "${JUNIT_DIR}"/junit_cluster_api_actuator_pkg_e2e_*.xml 1> /dev/null 2>&1 ; then
  echo "No files to merge"
  exit 0
fi

ensure::junitmerge

# If JUNIT_DIR is not set, no JUnit reports to merge
if [[ -z "${JUNIT_DIR:-}" ]]; then
	return
fi

output="$( mktemp )"
"$( find::junitmerge )" "${JUNIT_DIR}"/junit_cluster_api_actuator_pkg_e2e_*.xml > "${output}"
rm "${JUNIT_DIR}"/*.xml
mv "${output}" "${JUNIT_DIR}/junit_cluster_api_actuator_pkg_e2e.xml"

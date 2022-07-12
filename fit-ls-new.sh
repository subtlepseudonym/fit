#!/bin/bash

if [[ "$#" < 2 ]]; then
	echo "usage: ls-new.sh ACTIVITY_DIR SINCE"
fi

SOURCE="${1}"
SINCE="${2}"

since_unix="$(date --date "${SINCE}" +%s)"
for f in $(find "${SOURCE}" -type f -name '*.fit' | sort); do
	file_date="$(date --date "$(basename "${f}" | cut -d- -f1,2,3)" +%s)"
	if [[ "${since_unix}" -gt "${file_date}" ]]; then
		continue
	fi

	type="$(fit_type --data ${f})"
	if [[ "${type}" != "track" ]]; then
		echo "${f}" "${type}"
	fi
done

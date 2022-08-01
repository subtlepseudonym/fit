#!/bin/bash

wait_for_mount () {
	local mount_path="$1"
	local max_retries=60
	local retries=0

	while ! mountpoint --quiet "${mount_path}"; do
		retries=$(( ${retries} + 1 ))
		if [[ ${retries} > ${max_retries} ]]; then
			>&2 echo "max retry attempts exceeded waiting for mountpoint"
			exit 1
		fi
		sleep 1
	done

	if [[ 0 -ge "$(ls "${mount_path}" | wc -l)" ]]; then
		echo "no files on mount path"
		exit 1
	fi
}

import () {
	echo "importing $(basename $2)"
	/usr/bin/rsync \
		--stats \
		--info=name1 \
		--checksum \
		--no-perms \
		--chmod=-x \
		--owner \
		--group \
		--chown=1000:1000 \
		$1 \
		$2

	if [ $? != 0 ]; then
		>&2 echo "rsync failed"
		exit 1
	fi
	echo
}

remove_track_files () {
	local file_path="$1"
	for f in $(find "${file_path}" -type f -name '*.fit'); do
		type="$(fit type ${f})"
		if [[ "${type}" == "track" ]]; then
			echo "removing ${f}"
			rm $f
		fi
	done
}

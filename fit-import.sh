#!/bin/bash

GARMIN_MOUNT_PATH="$1"
DESTINATION_PATH="$2"

max_retries=60
retries=0
while ! mountpoint --quiet "${GARMIN_MOUNT_PATH}"; do
	retries=$(( $retries + 1 ))
	if [[ ${retries} > ${max_retries} ]]; then
		>&2 echo "max retry attempts exceeded waiting for mountpoint"
		exit 1
	fi
	sleep 1
done

if [[ 0 -ge "$(ls "${GARMIN_MOUNT_PATH}" | wc -l)" ]]; then
	echo "no files on mount path"
	exit 0
fi

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

# import fit files
import "${GARMIN_MOUNT_PATH}/GARMIN/Activity/*.fit" "${DESTINATION_PATH}/activity"
import "${GARMIN_MOUNT_PATH}/GARMIN/Monitor/*.FIT" "${DESTINATION_PATH}/monitor"

# remove tracking files from device
for f in $(find "${GARMIN_MOUNT_PATH}/GARMIN/Activity/" -type f -name '*.fit'); do
	type="$(fit type ${f})"
	if [[ "${type}" == "track" ]]; then
		echo "removing ${f}"
		rm $f
	fi
done

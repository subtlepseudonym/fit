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

if [[ 0 >= "$(ls "${GARMIN_MOUNT_PATH}" | wc -l)" ]]; then
	echo "no files to import"
	exit 0
fi

/usr/bin/rsync \
	--stats \
	--info=name1 \
	--no-perms \
	--chmod=-x \
	--owner \
	--group \
	--chown=1000:1000 \
	${GARMIN_MOUNT_PATH}/GARMIN/Activity/*.fit \
	"${DESTINATION_PATH}/activity"

if [ $? != 0 ]; then
	>&2 echo "rsync failed"
	exit 1
fi

for f in $(find "${GARMIN_MOUNT_PATH}/GARMIN/Activity/" -type f -name '*.fit'); do
	type="$(fit_type --data ${f})"
	if [[ "${type}" == "track" ]]; then
		echo "removing ${f}"
		rm $f
	fi
done

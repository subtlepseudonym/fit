#!/bin/bash

GARMIN_MOUNT_PATH="$1"
DESTINATION_PATH="$2"

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
		#rm $f
	fi
done

#!/bin/bash

if [[ "$#" < 2 ]]; then
	echo "usage: ls-new.sh ACTIVITY_DIR TOKEN_PATH"
fi

SOURCE="${1}"
TOKEN_PATH="${2}"
NEW_TOKEN_PATH="new_token.json"

{
	read access_token
	read refresh_token
	read expires_at
}< <(
	cat "${TOKEN_PATH}" \
	| jq --raw-output --compact-output '.access_token, .refresh_token, .expires_at'
)

if [[ "${expires_at}" -gt "$(date +%s)" ]]; then
	curl \
		--write-out "%{http_code}\n" \
		--request POST \
		--url "https://www.strava.com/api/v3/oauth/token" \
		--data client_id="${STRAVA_CLIENT_ID}" \
		--data client_secret="${STRAVA_CLIENT_SECRET}" \
		--data grant_type="refresh_token" \
		--data refresh_token="${STRAVA_REFRESH_TOKEN}" \
		1>"${NEW_TOKEN_PATH}" 2>/dev/null

	{
		read access_token
		read refresh_token
		read http_code
	}< <(
		jq \
			--raw-output \
			--compact-output \
			'def handle: .access_token, .refresh_token; . as $line | try handle catch $line' \
			"${NEW_TOKEN_PATH}"
	)

	if [[ "${http_code}" != "200" ]]; then
		echo "unexpected response code: ${http_code}"
		echo "see ${new_token_path}"
		exit 1
	fi
fi


latest="$(
	curl \
		--url "https://www.strava.com/api/v3/athlete/activities?per_page=5" \
		--header "Authentication: Bearer ${access_token}" \
	| jq --raw-output 'sort_by(.start_date) | .[-1] | .start_date'
)"
echo "latest on strava: ${latest}"

latest_unix="$(date -d @${latest} +%s)"
for f in $(find "${SOURCE}" -type f -name '*.fit' | sort); do
	file_date="$(date -d "$(basename "${f}" | cut -d- -f1,2,3)" +%s)"
	if [[ "${latest_unix}" -gt "${file_date}" ]]; then
		continue
	fi

	type="$(fit_type --data ${f})"
	if [[ "${type}" != "track" ]]; then
		echo "${f}" "${type}"
	fi
done

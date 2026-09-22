#!/usr/bin/env bash
set -euo pipefail

profile=${1:?usage: check-coverage.sh PROFILE MINIMUM}
minimum=${2:?usage: check-coverage.sh PROFILE MINIMUM}
go_cmd=${GO:-go}

total=$(
  "$go_cmd" tool cover -func="$profile" |
    awk '/^total:/ { sub("%", "", $3); print $3 }'
)

if [[ -z "$total" ]]; then
  echo "coverage profile did not contain a total: $profile" >&2
  exit 1
fi

echo "Total coverage: ${total}% (minimum ${minimum}%)"
if ! awk -v total="$total" -v minimum="$minimum" 'BEGIN { exit !(total >= minimum) }'; then
  echo "Coverage ${total}% is below the ${minimum}% minimum." >&2
  exit 1
fi

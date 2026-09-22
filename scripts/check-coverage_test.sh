#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

fake_go="$tmp_dir/go"
cat >"$fake_go" <<'EOF'
#!/usr/bin/env bash
printf 'total:\t(statements)\t%s%%\n' "${FAKE_COVERAGE:?}"
EOF
chmod +x "$fake_go"

output=$(FAKE_COVERAGE=90 GO="$fake_go" "$repo_root/scripts/check-coverage.sh" ignored.out 90)
[[ "$output" == *"Total coverage: 90% (minimum 90%)"* ]]

if FAKE_COVERAGE=89.9 GO="$fake_go" "$repo_root/scripts/check-coverage.sh" ignored.out 90 >"$tmp_dir/fail.out" 2>&1; then
  echo "expected coverage below the minimum to fail" >&2
  exit 1
fi
grep -F "Coverage 89.9% is below the 90% minimum." "$tmp_dir/fail.out" >/dev/null

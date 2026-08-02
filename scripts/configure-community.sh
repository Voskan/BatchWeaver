#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" != "--confirm-Voskan-BatchWeaver" ]; then
  printf 'Refusing GitHub mutation without --confirm-Voskan-BatchWeaver\n' >&2
  exit 2
fi
scripts/verify-github-release-gates.sh --repository-only
LABELS_TMP=$(mktemp)
trap 'rm -f "$LABELS_TMP"' EXIT
python3 - <<'PY' > "$LABELS_TMP"
import re
from pathlib import Path
pattern = re.compile(r'^- \{name: "([^"]+)", color: "([0-9a-f]+)", description: "([^"]+)"\}$')
for line in Path('.github/labels.yml').read_text().splitlines():
    match = pattern.fullmatch(line)
    if not match:
        raise SystemExit(f'invalid label definition: {line}')
    print(*match.groups(), sep='\t')
PY
while IFS=$'\t' read -r name color description; do
  if gh label list --repo Voskan/BatchWeaver --limit 100 --json name --jq '.[].name' | grep -Fxq "$name"; then
    gh label edit "$name" --repo Voskan/BatchWeaver --color "$color" --description "$description"
  else
    gh label create "$name" --repo Voskan/BatchWeaver --color "$color" --description "$description"
  fi
done < "$LABELS_TMP"
printf 'Labels configured; Discussions and private security reporting are enabled.\n'

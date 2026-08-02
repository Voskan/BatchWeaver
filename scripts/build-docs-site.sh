#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-_site}"
case "$OUTPUT" in
  ""|"/"|"."|"..") printf 'unsafe site output path\n' >&2; exit 2 ;;
esac

VERSION=$(tr -d '[:space:]' < release/VERSION)
COMMIT=$(git rev-parse HEAD)
BUILD_DATE=$(git show -s --format=%cs HEAD)
TEMP=$(mktemp -d)
trap 'rm -rf "$TEMP"' EXIT
mkdir -p "$TEMP/site"

while IFS= read -r -d '' bw_source; do
  bw_relative="${bw_source#site/}"
  bw_destination="$TEMP/site/$bw_relative"
  mkdir -p "$(dirname "$bw_destination")"
  case "$bw_source" in
    *.html|*.xml|*.txt|*.webmanifest)
      sed \
        -e "s/@@VERSION@@/$VERSION/g" \
        -e "s/@@COMMIT@@/$COMMIT/g" \
        -e "s/@@BUILD_DATE@@/$BUILD_DATE/g" \
        "$bw_source" > "$bw_destination"
      ;;
    *) cp "$bw_source" "$bw_destination" ;;
  esac
done < <(find site -type f -print0 | sort -z)

test -s "$TEMP/site/index.html"
test -s "$TEMP/site/styles.css"
test -s "$TEMP/site/docs.html"
test -s "$TEMP/site/examples.html"
test -s "$TEMP/site/api.html"
test -s "$TEMP/site/status.html"
if rg -n '@@VERSION@@|@@COMMIT@@|localhost|/Users/|\.agent/' "$TEMP/site"; then
  printf 'site contains an unresolved or private reference\n' >&2
  exit 1
fi
rm -rf "$OUTPUT"
mv "$TEMP/site" "$OUTPUT"
printf 'Documentation site built at %s for %s (%s)\n' "$OUTPUT" "$VERSION" "$COMMIT"

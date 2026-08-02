#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-_site}"
case "$OUTPUT" in
  ""|"/"|"."|"..") printf 'unsafe site output path\n' >&2; exit 2 ;;
esac

VERSION=$(tr -d '[:space:]' < release/VERSION)
COMMIT=$(git rev-parse HEAD)
TEMP=$(mktemp -d)
trap 'rm -rf "$TEMP"' EXIT
mkdir -p "$TEMP/site"
sed -e "s/@@VERSION@@/$VERSION/g" -e "s/@@COMMIT@@/$COMMIT/g" site/index.html > "$TEMP/site/index.html"
cp site/styles.css site/favicon.svg site/robots.txt site/sitemap.xml "$TEMP/site/"
test -s "$TEMP/site/index.html"
test -s "$TEMP/site/styles.css"
if rg -n '@@VERSION@@|@@COMMIT@@|localhost|/Users/|\.agent/' "$TEMP/site"; then
  printf 'site contains an unresolved or private reference\n' >&2
  exit 1
fi
rm -rf "$OUTPUT"
mv "$TEMP/site" "$OUTPUT"
printf 'Documentation site built at %s for %s (%s)\n' "$OUTPUT" "$VERSION" "$COMMIT"

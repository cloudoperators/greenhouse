#!/bin/sh

set -e
/usr/local/bin/generate_manifest.sh --manifest=/usr/share/nginx/html/manifest.json --apps=/usr/share/nginx/html/apps --extensions=/usr/share/nginx/html/extensions
/docker-entrypoint.sh

exec "$@"

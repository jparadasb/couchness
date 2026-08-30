#!/bin/sh
set -eu

couchness --config-dir /config telegram setup --owner-id "${COUCHNESS_TELEGRAM_OWNER_ID}"
exec couchness --config-dir /config telegram run

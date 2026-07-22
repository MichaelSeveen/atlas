#!/bin/sh
set -eu

[ -d /backups ]
[ -d /wal-archive ]
chown postgres:postgres /backups /wal-archive
chmod 0700 /backups /wal-archive
exec docker-entrypoint.sh "$@"

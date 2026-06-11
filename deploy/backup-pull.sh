#!/bin/sh
# Offsite pull of the VPS fund.db to this Mac, run by launchd twice a day.
#
# Lives (installed) at ~/Library/FundBackups/pull.sh — NOT run from the
# project dir, because launchd agents cannot read TCC-protected folders
# (~/Desktop) without Full Disk Access. Same reason the key is a copy at
# ~/.ssh/tokyo-ali.pem and snapshots land in ~/Library/FundBackups/snapshots.
#
# Install / update (from project root):
#   cp deploy/backup-pull.sh ~/Library/FundBackups/pull.sh && chmod +x ~/Library/FundBackups/pull.sh
#   cp ~/Desktop/pem/tokyo-ali.pem ~/.ssh/tokyo-ali.pem && chmod 600 ~/.ssh/tokyo-ali.pem
#   cp deploy/backup-launchd.plist ~/Library/LaunchAgents/top.xg22.fund-backup.plist
#   launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/top.xg22.fund-backup.plist
#
# For ad-hoc pulls into the project's ./backups dir, use deploy/backup.sh.
set -eu

KEY=$HOME/.ssh/tokyo-ali.pem
HOST=root@47.245.31.99
VPS_DB=/root/stacks/dashboard/data/fund.db
OUT_DIR=$HOME/Library/FundBackups/snapshots

TS=$(date -u +%Y%m%dT%H%M%SZ)
DEST=$OUT_DIR/fund.db.vps-$TS
mkdir -p "$OUT_DIR"

# Online .backup on the VPS — safe against the running dashboard (WAL).
ssh -i "$KEY" -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15 "$HOST" \
  "sqlite3 $VPS_DB \".backup /tmp/fund-snapshot.db\" && chmod 600 /tmp/fund-snapshot.db"
scp -q -i "$KEY" "$HOST:/tmp/fund-snapshot.db" "$DEST"
ssh -i "$KEY" "$HOST" "rm -f /tmp/fund-snapshot.db"

# immutable=1 keeps the check from spawning -wal/-shm sidecars locally.
if ! sqlite3 "file:$DEST?immutable=1" 'PRAGMA integrity_check;' | grep -q '^ok$'; then
  echo "$(date -u +%FT%TZ) integrity_check FAILED on $DEST — keeping file for inspection" >&2
  exit 1
fi

# Keep the last 60 snapshots (= 30 days at 2/day).
ls -1t "$OUT_DIR"/fund.db.vps-* 2>/dev/null | tail -n +61 | while read -r f; do rm -f "$f"; done

echo "$(date -u +%FT%TZ) ok $DEST"

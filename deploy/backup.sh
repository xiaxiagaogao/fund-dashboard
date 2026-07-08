#!/usr/bin/env bash
#
# Pull a WAL-consistent snapshot of the production fund.db down to local
# ./backups/. Run as often as you like — sqlite3's .backup is safe to call
# against a running database (it uses online backup API).
#
# Usage:
#   ./deploy/backup.sh                  # one-shot, timestamped filename
#   ./deploy/backup.sh --latest         # also overwrite ./data/fund.db (DESTRUCTIVE — for restore)
#
# Backup layout:
#   * VPS-side (automated): /root/stacks/dashboard/backup-local.sh via root
#     crontab, every 6h, keeps ~7 days. Guards against corruption/mistakes.
#   * Mac-side (manual, this script): run every few days to keep an offsite
#     copy in ./backups in case the VPS disk dies. Keeps last 30.
set -euo pipefail

# Connection details come from deploy/deploy.env (gitignored). Environment
# variables still override, e.g. VPS_HOST=1.2.3.4 ./deploy/backup.sh.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ -f "$SCRIPT_DIR/deploy.env" ]; then
    # shellcheck disable=SC1091
    source "$SCRIPT_DIR/deploy.env"
fi

cd "$(dirname "$0")/.."

VPS_USER=${VPS_USER:-root}
VPS_HOST=${VPS_HOST:?set VPS_HOST in deploy/deploy.env or the environment}
SSH_KEY=${SSH_KEY:?set SSH_KEY in deploy/deploy.env or the environment}
VPS_DB=${VPS_DB:-/root/stacks/dashboard/data/fund.db}

mkdir -p backups
TS=$(date -u +%Y%m%dT%H%M%SZ)
DEST=backups/fund.db.vps-$TS

# 1. Online .backup on VPS — safe vs the running dashboard (WAL).
ssh -i "$SSH_KEY" "$VPS_USER@$VPS_HOST" \
  "sqlite3 $VPS_DB \".backup /tmp/fund-snapshot.db\" && chmod 600 /tmp/fund-snapshot.db"

# 2. Pull it down.
scp -q -i "$SSH_KEY" "$VPS_USER@$VPS_HOST:/tmp/fund-snapshot.db" "$DEST"

# 3. Wipe the VPS-side temp.
ssh -i "$SSH_KEY" "$VPS_USER@$VPS_HOST" "rm -f /tmp/fund-snapshot.db"

# 4. Sanity-check the local copy. Use immutable=1 so the check itself doesn't
# spawn WAL/SHM sidecar files in backups/ (keeps the dir tidy).
if ! sqlite3 "file:$DEST?immutable=1" 'PRAGMA integrity_check;' | grep -q '^ok$'; then
  echo "❌ integrity_check FAILED on $DEST — keeping file for inspection" >&2
  exit 1
fi
echo "✅ $DEST  ($(du -h "$DEST" | cut -f1))"

# Roll: keep last 30 main snapshots, prune older (incl. any stray wal/shm).
ls -1t backups/fund.db.vps-*[!ml] 2>/dev/null | tail -n +31 | while read -r f; do
  rm -f "$f" "$f-wal" "$f-shm"
done

# --latest: also overwrite ./data/fund.db so a local server can serve current state
if [ "${1:-}" = "--latest" ]; then
  mkdir -p data
  cp -p "$DEST" data/fund.db
  rm -f data/fund.db-wal data/fund.db-shm    # the .backup output is a single-file snapshot
  echo "   also copied → ./data/fund.db (local dev now serves current prod state)"
fi

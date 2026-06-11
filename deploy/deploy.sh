#!/usr/bin/env bash
# Deploy fund-dashboard to the nofx VPS.
#
# Usage (from project root):
#
#   ./deploy/deploy.sh               # rsync + build + restart (never touches VPS fund.db)
#   ./deploy/deploy.sh --build-only  # rsync + build, don't restart
#   ./deploy/deploy.sh --push-db     # DANGER: overwrite VPS fund.db with local ./data/fund.db.
#                                    # The VPS is the source of truth (live ledger writes) —
#                                    # only for disaster recovery from a verified backup.
#                                    # Snapshots the VPS DB and stops the container first.
#
# Assumptions:
#   - SSH key at $HOME/Desktop/pem/tokyo-ali.pem reaches root@47.245.31.99
#   - Local .env has Binance keys (gets copied to VPS as-is, then VPS .env
#     gets JWT_SECRET overwritten with a fresh random value the first time)
#   - Cloudflare DNS for fund.xg22.top → 47.245.31.99 already in place
set -euo pipefail

VPS_HOST="root@47.245.31.99"
SSH_KEY="$HOME/Desktop/pem/tokyo-ali.pem"
SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=accept-new"
RSYNC_SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=accept-new"

REMOTE_SRC="/root/src/fund-dashboard"
REMOTE_STACK="/root/stacks/dashboard"

PUSH_DB=0
BUILD_ONLY=0
for arg in "$@"; do
    case "$arg" in
        --push-db)    PUSH_DB=1 ;;
        --no-db)      echo "note: --no-db is now the default; use --push-db to overwrite the VPS DB" ;;
        --build-only) BUILD_ONLY=1 ;;
        *) echo "unknown flag: $arg" >&2; exit 2 ;;
    esac
done

cd "$(dirname "$0")/.."

echo "==> 1/6  Probing VPS reachability"
$SSH $VPS_HOST "echo ok && uname -m && docker --version" || { echo "VPS unreachable"; exit 1; }

echo "==> 2/6  Ensuring remote dirs exist"
$SSH $VPS_HOST "mkdir -p $REMOTE_SRC $REMOTE_STACK/data $REMOTE_STACK/nginx"

echo "==> 3/6  Rsync source (skip node_modules, .svelte-kit, build, data, .env*)"
rsync -az --delete \
    -e "$RSYNC_SSH" \
    --exclude='.git/' \
    --exclude='.DS_Store' \
    --exclude='web/node_modules/' \
    --exclude='web/.svelte-kit/' \
    --exclude='web/build/' \
    --exclude='data/' \
    --exclude='.env' \
    --exclude='.env.real' \
    ./  $VPS_HOST:$REMOTE_SRC/

echo "==> 4/6  Place / refresh stack files (docker-compose, nginx vhost, .env)"
# docker-compose.yml + Dockerfile go into the stack dir alongside data/
$SSH $VPS_HOST "cp -f $REMOTE_SRC/docker-compose.yml $REMOTE_STACK/docker-compose.yml \
             && cp -f $REMOTE_SRC/Dockerfile         $REMOTE_STACK/Dockerfile"

# Create or refresh .env on VPS. On first deploy, generate a fresh
# JWT_SECRET and lift BINANCE_* keys from the local .env so dashboard
# starts with a working config in one shot. Subsequent deploys leave
# the existing .env alone.
LOCAL_BN_KEY=$(grep -E '^BINANCE_API_KEY=' .env 2>/dev/null | cut -d= -f2- || true)
LOCAL_BN_SECRET=$(grep -E '^BINANCE_API_SECRET=' .env 2>/dev/null | cut -d= -f2- || true)
if [ -z "$LOCAL_BN_KEY" ] || [ -z "$LOCAL_BN_SECRET" ]; then
    echo "WARN: local .env missing BINANCE_API_KEY/SECRET — first deploy may need a manual fill on VPS." >&2
fi

$SSH $VPS_HOST "bash -s" <<EOF_REMOTE
set -e
ENV_PATH="$REMOTE_STACK/.env"
if [ ! -f "\$ENV_PATH" ]; then
    JWT=\$(openssl rand -base64 32 | tr -d '\n=' | head -c 44)
    cat > "\$ENV_PATH" <<ENV
# Auto-generated on first deploy by deploy.sh.
# JWT_SECRET is unique to this VPS and not stored anywhere else.
BINANCE_API_KEY=$LOCAL_BN_KEY
BINANCE_API_SECRET=$LOCAL_BN_SECRET
JWT_SECRET=\$JWT
HTTP_ADDR=:8090
ENV
    chmod 600 "\$ENV_PATH"
    echo "  wrote fresh \$ENV_PATH (chmod 600). JWT_SECRET length=\${#JWT} (value not echoed)"
else
    echo "  reusing existing \$ENV_PATH"
fi
EOF_REMOTE

# Sync the dashboard's source-of-truth source files into the stack so
# `docker compose build` can use them.
$SSH $VPS_HOST "cp -rf $REMOTE_SRC/* $REMOTE_STACK/ && rm -rf $REMOTE_STACK/deploy" || true

if [ $PUSH_DB -eq 1 ]; then
    echo "==> 5/6  --push-db: about to OVERWRITE the live VPS ledger with the local copy"
    echo "    local : $(ls -l ./data/fund.db)"
    echo "    remote: $($SSH $VPS_HOST "ls -l $REMOTE_STACK/data/fund.db 2>/dev/null || echo '(absent)'")"
    read -r -p "    Type 'overwrite' to continue: " CONFIRM
    if [ "$CONFIRM" != "overwrite" ]; then
        echo "aborted — VPS fund.db untouched."
        exit 1
    fi
    # Snapshot the current VPS DB, stop the writer, then replace. Stale WAL/SHM
    # sidecars belong to the old DB and must not be replayed into the new one.
    $SSH $VPS_HOST "/root/stacks/dashboard/backup-local.sh"
    $SSH $VPS_HOST "cd $REMOTE_STACK && docker compose stop dashboard"
    rsync -az -e "$RSYNC_SSH" ./data/fund.db $VPS_HOST:$REMOTE_STACK/data/fund.db
    $SSH $VPS_HOST "rm -f $REMOTE_STACK/data/fund.db-wal $REMOTE_STACK/data/fund.db-shm"
    echo "    pushed; container restarts in step 6 (with --build-only it stays STOPPED)"
else
    echo "==> 5/6  Leaving VPS fund.db untouched (default; --push-db to overwrite)"
fi

if [ $BUILD_ONLY -eq 1 ]; then
    echo "==> 6/6  Build only — running 'docker compose build' on VPS"
    $SSH $VPS_HOST "cd $REMOTE_STACK && docker compose build"
    exit 0
fi

echo "==> 6/6  Build + restart on VPS"
$SSH $VPS_HOST "cd $REMOTE_STACK && docker compose up -d --build"

echo
echo "==> Post-deploy probe"
$SSH $VPS_HOST "sleep 3 && docker compose -f $REMOTE_STACK/docker-compose.yml ps && curl -sI http://127.0.0.1:8090/healthz | head -1"

cat <<'POST'

Manual follow-ups:

  1. nginx vhost (one time, then cached):
       scp -i $HOME/Desktop/pem/tokyo-ali.pem nginx/fund.xg22.top.conf root@47.245.31.99:/etc/nginx/conf.d/
       ssh -i $HOME/Desktop/pem/tokyo-ali.pem root@47.245.31.99 'nginx -t && nginx -s reload'

  2. Cloudflare DNS (one time):
       Add A record  fund.xg22.top → 47.245.31.99  (Proxied = orange cloud)

  3. Verify external:
       curl -sI https://fund.xg22.top/healthz

  4. Make sure nofx is still healthy:
       ssh root@47.245.31.99 'docker logs nofx-trading --tail 5 && docker stats --no-stream'
POST

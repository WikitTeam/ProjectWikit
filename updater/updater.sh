#!/usr/bin/env bash
set -uo pipefail

MAILBOX="${MAILBOX:-/mailbox}"
REPO="${UPDATE_REPO:-WikitTeam/ProjectWikit}"
BRANCH="${UPDATE_BRANCH:-master}"
POLL="${UPDATE_POLL_INTERVAL:-600}"
PROJECT_DIR="${HOST_PROJECT_DIR:?HOST_PROJECT_DIR is required}"

REQ="$MAILBOX/update.request"
REQ_LOCK="$MAILBOX/update.request.processing"
STATUS="$MAILBOX/update.status"
LOG="$MAILBOX/update.log"
RELEASE_CACHE="$MAILBOX/release_cache.json"
DEPLOYED="$MAILBOX/deployed.json"

mkdir -p "$MAILBOX"
chmod 0777 "$MAILBOX" 2>/dev/null || true
git config --global --add safe.directory "$PROJECT_DIR" 2>/dev/null || true

now() { date -u +%FT%TZ; }

write_status() {
  jq -n --arg state "$1" --arg msg "$2" --arg started "${STARTED:-}" --arg finished "${3:-}" \
    '{state:$state, message:$msg, started_at:$started, finished_at:$finished}' \
    > "$STATUS.tmp" && mv "$STATUS.tmp" "$STATUS"
  chmod 0666 "$STATUS" 2>/dev/null || true
}

write_deployed() {
  cd "$PROJECT_DIR" || return
  local ref sha
  ref="$(git describe --tags --always 2>/dev/null || echo unknown)"
  sha="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
  jq -n --arg ref "$ref" --arg sha "$sha" --arg at "$(now)" \
    '{ref:$ref, sha:$sha, at:$at}' > "$DEPLOYED.tmp" && mv "$DEPLOYED.tmp" "$DEPLOYED"
  chmod 0666 "$DEPLOYED" 2>/dev/null || true
}

poll_github() {
  local resp
  resp="$(curl -sf -H 'Accept: application/vnd.github+json' \
      "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null)" || return 0
  [ -z "$resp" ] && return 0
  echo "$resp" | jq --arg now "$(now)" \
    '{tag: .tag_name, name: .name, body: .body, html_url: .html_url, published_at: .published_at, fetched_at: $now}' \
    > "$RELEASE_CACHE.tmp" 2>/dev/null && mv "$RELEASE_CACHE.tmp" "$RELEASE_CACHE"
  chmod 0666 "$RELEASE_CACHE" 2>/dev/null || true
}

do_update() {
  STARTED="$(now)"
  : > "$LOG"; chmod 0666 "$LOG" 2>/dev/null || true
  local target checkout_target
  target="$(tr -d '[:space:]' < "$REQ_LOCK" 2>/dev/null)"

  log() { echo "[$(now)] $*" | tee -a "$LOG"; }

  write_status running "开始更新..."
  cd "$PROJECT_DIR" || { write_status failed "找不到项目目录 $PROJECT_DIR" "$(now)"; return; }

  log "拉取远程 tags..."
  if ! git fetch --tags --force origin >>"$LOG" 2>&1; then
    write_status failed "git fetch 失败，见日志" "$(now)"; return
  fi

  checkout_target="$target"
  [ -z "$checkout_target" ] && checkout_target="origin/$BRANCH"

  log "检出 $checkout_target ..."
  if ! git checkout "$checkout_target" >>"$LOG" 2>&1; then
    write_status failed "检出失败：服务器上可能存在未提交的本地改动，已中止（未覆盖任何文件）。详见日志。" "$(now)"
    return
  fi

  log "重建并重启 web 容器..."
  if ! docker compose up -d --no-deps --build web >>"$LOG" 2>&1; then
    write_status failed "docker compose 构建/启动失败，见日志" "$(now)"; return
  fi

  log "清理悬空镜像与旧构建缓存..."
  docker image prune -f >>"$LOG" 2>&1 || true
  docker builder prune -f --filter until=168h >>"$LOG" 2>&1 || true

  write_deployed
  log "更新完成。"
  write_status success "更新完成" "$(now)"
}

write_deployed
poll_github

last_poll="$(date +%s)"
while true; do
  if [ -f "$REQ" ]; then
    mv -f "$REQ" "$REQ_LOCK" 2>/dev/null || true
    do_update
    rm -f "$REQ_LOCK" 2>/dev/null || true
  fi
  nowsec="$(date +%s)"
  if [ $(( nowsec - last_poll )) -ge "$POLL" ]; then
    poll_github
    last_poll="$nowsec"
  fi
  sleep 5
done

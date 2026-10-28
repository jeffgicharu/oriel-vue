#!/usr/bin/env bash
#
# Puts an uploaded release live. Installed as /usr/local/bin/oriel-vue-activate.
#
# The deploy pipeline reaches this through an SSH key whose only permitted
# command is this script, so the one thing it can ask for is "activate the
# release with this commit id".
#
# A release runs as a pair: the web server that renders the pages, and the API
# it reads from. There are two slots, blue (web 8700, API 8702) and green (web
# 8701, API 8703), and nginx sends visitors to one of them. A new release is
# started in the idle slot, beside the live one, and must pass its check
# there: every page renders with its title, description, canonical link and
# structured data, and the API books a viewing, refuses a bad form with field
# errors, requests a reservation and has the sales desk confirm it. Only then
# do visitors reach it. The pages, the API and a live-update stream are then
# checked again through nginx; if they fail, nginx goes back to the release
# before, which has been running all along.
#
# To go back by hand, run this script with the commit id of an earlier
# release that is still in the releases directory.

set -euo pipefail

BASE=/var/www/oriel-vue.jeffgicharu.com
HOST=oriel-vue.jeffgicharu.com
WEB_USER=oriel-vue-web
API_USER=oriel-vue-api
API_ENV=$BASE/shared/api.env
STATIC=$BASE/shared/static
NGINX_DIR=/etc/nginx/oriel-vue
KEEP_RELEASES=3

release_id="${SSH_ORIGINAL_COMMAND:-${1:-}}"

if [[ ! "$release_id" =~ ^[0-9a-f]{40}$ ]]; then
    echo "Expected a 40 character commit id." >&2
    exit 64
fi

release="$BASE/releases/$release_id"

for file in REVISION unavailable.html api/oriel-vue-api node/bin/node web/server.mjs web/client/index.html; do
    if [[ ! -f "$release/$file" ]]; then
        echo "Release $release_id has not been uploaded completely ($file is missing)." >&2
        exit 66
    fi
done
if [[ "$(cat "$release/REVISION")" != "$release_id" ]]; then
    echo "Release $release_id carries the wrong REVISION." >&2
    exit 66
fi

# One deploy at a time.
exec 9>"$BASE/.activate.lock"
flock -n 9 || { echo "Another activation is running." >&2; exit 75; }

web_port() { case "$1" in blue) echo 8700 ;; green) echo 8701 ;; esac; }
api_port() { case "$1" in blue) echo 8702 ;; green) echo 8703 ;; esac; }

live_web="$(grep -oE '127\.0\.0\.1:870[01]' "$NGINX_DIR/web.conf" | head -n1 | cut -d: -f2 || true)"
case "$live_web" in
    8700) live=blue; idle=green ;;
    8701) live=green; idle=blue ;;
    *) echo "Cannot tell which slot is live from $NGINX_DIR/web.conf." >&2; exit 70 ;;
esac
idle_web="$(web_port "$idle")"
idle_api="$(api_port "$idle")"

live_release="$(readlink -f "$BASE/slots/$live" 2>/dev/null || true)"
first_release=""
if [[ -z "$live_release" || ! -d "$live_release" ]] \
    || ! systemctl is-active --quiet "oriel-vue-web@$live.service" \
    || ! systemctl is-active --quiet "oriel-vue-api@$live.service"; then
    first_release=yes
fi

# Asked again for the release that is already live: nothing to do.
if [[ -z "$first_release" && "$live_release" == "$release" ]]; then
    echo "Release ${release_id:0:7} is already live in the $live slot."
    exit 0
fi

# nginx is shared by every site on this server. If its configuration is
# already broken by something else, nothing here should touch it.
if ! sudo nginx -t >/dev/null 2>&1; then
    echo "nginx's configuration does not pass its own test before this release changed anything." >&2
    echo "Nothing was changed. Find out which site broke it before deploying again." >&2
    exit 70
fi

chmod 755 "$release/api/oriel-vue-api" "$release/node/bin/node"

# Release code only ever runs as one of the application users. This script
# runs as deploy, which can become root, so it never executes anything it was
# sent as itself. Neither command reads the data folder or the working
# directory: version prints the build's commit id, and the check reads only
# the settings file it is given and talks to the slot over 127.0.0.1.
as_api() {
    ( cd / && sudo -u "$API_USER" -- "$release/api/oriel-vue-api" "$@" )
}
as_web() {
    ( cd "$release/web" && sudo -u "$WEB_USER" -- "$release/node/bin/node" server.mjs "$@" )
}

# The processor here is old. Prove both halves of this build run on it before
# anything else.
if ! reported="$(as_api version 2>&1)" || [[ "$reported" != "$release_id" ]]; then
    echo "The API of release ${release_id:0:7} does not run on this server: $reported" >&2
    exit 1
fi
if ! reported="$(as_web version 2>&1)" || [[ "$reported" != "$release_id" ]]; then
    echo "The web server of release ${release_id:0:7} does not run on this server: $reported" >&2
    exit 1
fi

# Fingerprinted files arrive as plain files in the release. A name never
# changes its contents, so files already shared are left as they are.
echo "Sharing this release's scripts, styles, fonts and pictures"
mkdir -p "$STATIC/assets"
rsync -a --ignore-existing "$release/web/client/assets/" "$STATIC/assets/"
install -m 644 "$release/unavailable.html" "$BASE/shared/pages/unavailable.html"

echo "Starting release ${release_id:0:7} in the $idle slot (web $idle_web, API $idle_api)"
sudo systemctl stop "oriel-vue-web@$idle.service" "oriel-vue-api@$idle.service" 2>/dev/null || true
ln -sfn "$release" "$BASE/slots/$idle.new"
mv -Tf "$BASE/slots/$idle.new" "$BASE/slots/$idle"
sudo systemctl start "oriel-vue-api@$idle.service"
sudo systemctl start "oriel-vue-web@$idle.service"

abandon() {
    sudo systemctl stop "oriel-vue-web@$idle.service" "oriel-vue-api@$idle.service" 2>/dev/null || true
    echo "$1" >&2
    exit 1
}

show_logs() {
    sudo journalctl -u "oriel-vue-api@$idle.service" -n 20 --no-pager >&2 || true
    sudo journalctl -u "oriel-vue-web@$idle.service" -n 20 --no-pager >&2 || true
}

# Both halves report the release they run. The web server's answer also says
# it can reach its own API.
answers() {
    curl --silent --max-time 3 --head "$1" 2>/dev/null | grep -qi "^x-release: $release_id"
}
started=""
for attempt in $(seq 1 40); do
    if answers "http://127.0.0.1:$idle_api/api/healthz" && answers "http://127.0.0.1:$idle_web/healthz"; then
        started=yes
        break
    fi
    sleep 1
done
if [[ -z "$started" ]]; then
    show_logs
    abandon "Release ${release_id:0:7} did not start. The site is unchanged."
fi

# The gate. Visitors are still on the release before, so a release that fails
# here never reaches them. The check reads the API's settings file itself, for
# the sales desk's sign-in.
echo "Checking the release before it goes live"
if ! as_api check-release --env-file "$API_ENV" --web "http://127.0.0.1:$idle_web" --api "http://127.0.0.1:$idle_api" --release "$release_id"; then
    show_logs
    abandon "Release ${release_id:0:7} failed its check and was not put live."
fi

point_nginx_at() {
    local slot="$1"
    printf 'server 127.0.0.1:%s;\n' "$(web_port "$slot")" | sudo tee "$NGINX_DIR/web.conf.new" >/dev/null
    printf 'server 127.0.0.1:%s;\n' "$(api_port "$slot")" | sudo tee "$NGINX_DIR/api.conf.new" >/dev/null
    sudo mv -f "$NGINX_DIR/web.conf.new" "$NGINX_DIR/web.conf"
    sudo mv -f "$NGINX_DIR/api.conf.new" "$NGINX_DIR/api.conf"
    sudo nginx -t >/dev/null 2>&1 || return 1
    sudo systemctl reload nginx
}

# Asks the web server on this machine for a page under the public name. The
# certificate is the one Cloudflare trusts, not one curl knows, hence --insecure.
fetch() {
    curl --silent --show-error --insecure --max-time 20 --resolve "$HOST:443:127.0.0.1" "$@"
}

serves_the_release() {
    local attempt paths path headers stream
    for attempt in 1 2 3 4 5 6 7 8 9 10; do
        if fetch --head "https://$HOST/healthz" 2>/dev/null | grep -qi "^x-release: $release_id" \
            && fetch --head "https://$HOST/api/healthz" 2>/dev/null | grep -qi "^x-release: $release_id"; then
            break
        fi
        if [[ $attempt -eq 10 ]]; then
            echo "The web server is not serving ${release_id:0:7}." >&2
            return 1
        fi
        sleep 1
    done
    # The release names its own public pages and API reads. Each must load and
    # still ask search engines not to list it.
    paths="$(as_api check-release --print-paths)" || return 1
    while IFS= read -r path; do
        [[ -z "$path" ]] && continue
        if ! headers="$(fetch --fail --dump-header - --output /dev/null "https://$HOST$path")"; then
            echo "$path did not load on the live site." >&2
            return 1
        fi
        if ! grep -qi '^x-robots-tag: noindex, nofollow' <<<"$headers"; then
            echo "$path is missing its noindex header on the live site." >&2
            return 1
        fi
    done <<<"$paths"
    # A live-update stream through nginx opens with a retry line. The stream
    # stays open, so curl is cut off after a few seconds and only its output counts.
    stream="$(fetch --max-time 4 "https://$HOST/api/events" 2>/dev/null || true)"
    if ! grep -q '^retry:' <<<"$stream"; then
        echo "Live updates do not get through nginx." >&2
        return 1
    fi
}

go_back() {
    if [[ -n "$first_release" ]]; then
        abandon "$1 There was no release before this one to go back to."
    fi
    if point_nginx_at "$live"; then
        abandon "$1 Visitors are back on the release before."
    fi
    abandon "$1 nginx could not be pointed back. Look at this now."
}

echo "Switching visitors to release ${release_id:0:7}"
if ! point_nginx_at "$idle"; then
    # Put things back as they were. nginx was not reloaded.
    printf 'server 127.0.0.1:%s;\n' "$(web_port "$live")" | sudo tee "$NGINX_DIR/web.conf" >/dev/null
    printf 'server 127.0.0.1:%s;\n' "$(api_port "$live")" | sudo tee "$NGINX_DIR/api.conf" >/dev/null
    abandon "nginx refused its configuration. The site is unchanged."
fi

echo "Checking the live site"
if ! serves_the_release; then
    go_back "Release ${release_id:0:7} did not serve its pages."
fi

# The slot that is live is the one that starts when the server boots.
sudo systemctl enable "oriel-vue-api@$idle.service" "oriel-vue-web@$idle.service" >/dev/null 2>&1 || true
sudo systemctl disable "oriel-vue-api@$live.service" "oriel-vue-web@$live.service" >/dev/null 2>&1 || true

# Requests in hand on the old release are given time to finish. Stopping its
# API ends its live-update streams, and those browsers reconnect to the new one.
sleep 10
sudo systemctl stop "oriel-vue-web@$live.service" "oriel-vue-api@$live.service" 2>/dev/null || true

echo "Removing old releases"
cd "$BASE/releases"
ls -1t | tail -n +$((KEEP_RELEASES + 1)) | while read -r old; do
    [[ "$old" =~ ^[0-9a-f]{40}$ && "$old" != "$release_id" && "$BASE/releases/$old" != "$live_release" ]] || continue
    rm -rf -- "${BASE:?}/releases/$old"
done

# Fingerprinted files are kept for every release still on disk.
keep="$(mktemp)"
for kept in "$BASE"/releases/*/; do
    [[ -d "$kept/web/client/assets" ]] && ( cd "$kept/web/client/assets" && find . -type f ) >>"$keep"
done
sort -u "$keep" -o "$keep"
if [[ -s "$keep" && -d "$STATIC/assets" ]]; then
    ( cd "$STATIC/assets" && find . -type f ) | sort | comm -23 - "$keep" | while read -r stale; do
        rm -f -- "$STATIC/assets/$stale"
    done
    find "$STATIC/assets" -mindepth 1 -type d -empty -delete 2>/dev/null || true
fi
rm -f "$keep"

echo "Release ${release_id:0:7} is live in the $idle slot (web $idle_web, API $idle_api)."

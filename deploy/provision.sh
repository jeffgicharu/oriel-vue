#!/usr/bin/env bash
#
# Prepares the server for Oriel's Vue site. Run once, as deploy, from a
# temporary folder (made with mktemp -d) that holds the files it installs:
#
#   nginx.conf  oriel-vue-web@.service  oriel-vue-api@.service  activate.sh
#   unavailable.html  deploy-keys.pub  desk-entry-key
#
# The folder is sent and unpacked in the same SSH connection that runs this:
#
#   tar -czf - -C <folder> . | ssh deploy@<server> \
#     'd="$(mktemp -d)" && { tar -xzf - -C "$d" && bash "$d/provision.sh"; rc=$?; rm -rf -- "$d"; exit $rc; }'
#
# The folder holds a key, so it is removed when this script ends, whether it
# finishes or fails, and again by the command above if unpacking fails. It is
# safe to run again: anything that already exists is left alone, and no
# secret is ever printed.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
BASE=/var/www/oriel-vue.jeffgicharu.com
SITE=https://oriel-vue.jeffgicharu.com

case "$HERE" in
    /tmp/tmp.*) trap 'rm -rf -- "$HERE"' EXIT ;;
    *) echo "Run this from a folder made by mktemp -d, so it can be removed afterwards." >&2; exit 64 ;;
esac

step() { printf '\n== %s\n' "$1"; }

step "Application users"
# The web server renders pages and has no access to the data. The API owns it.
for user in oriel-vue-web oriel-vue-api; do
    if id "$user" >/dev/null 2>&1; then
        echo "$user exists"
    else
        sudo useradd --system --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin "$user"
        echo "created $user"
    fi
done

step "Folders"
sudo install -d -o deploy -g deploy -m 755 "$BASE" "$BASE/releases" "$BASE/slots" "$BASE/shared" "$BASE/shared/static" "$BASE/shared/pages"
# The databases: the shared sample and every visitor's copy. Only the API reads or writes here.
sudo install -d -o oriel-vue-api -g oriel-vue-api -m 750 "$BASE/shared/data"
install -m 644 "$HERE/unavailable.html" "$BASE/shared/pages/unavailable.html"
ls -ld "$BASE" "$BASE"/* "$BASE"/shared/*

step "Slot ports"
# Each slot is a web server and the API it reads from.
printf 'ORIEL_WEB_PORT=8700\nORIEL_API_PORT=8702\n' | sudo tee "$BASE/shared/blue.ports" >/dev/null
printf 'ORIEL_WEB_PORT=8701\nORIEL_API_PORT=8703\n' | sudo tee "$BASE/shared/green.ports" >/dev/null
sudo chown root:root "$BASE/shared/blue.ports" "$BASE/shared/green.ports"
sudo chmod 644 "$BASE/shared/blue.ports" "$BASE/shared/green.ports"
cat "$BASE/shared/blue.ports" "$BASE/shared/green.ports"

step "Settings"
if sudo test -f "$BASE/shared/api.env"; then
    echo "API settings file exists; left as it is"
else
    desk_key="$(tr -d '\n' < "$HERE/desk-entry-key")"
    # Created with its final owner and mode before anything is written into it.
    sudo install -m 640 -o root -g oriel-vue-api /dev/null "$BASE/shared/api.env"
    {
        echo "ORIEL_SITE_URL=$SITE"
        echo "ORIEL_SECRET=$(openssl rand -hex 32)"
        echo "ORIEL_DESK_KEY=$desk_key"
    } | sudo tee -a "$BASE/shared/api.env" >/dev/null
    unset desk_key
    echo "wrote the API settings file"
fi
if sudo test -f "$BASE/shared/web.env"; then
    echo "web settings file exists; left as it is"
else
    sudo install -m 640 -o root -g oriel-vue-web /dev/null "$BASE/shared/web.env"
    echo "ORIEL_SITE_URL=$SITE" | sudo tee -a "$BASE/shared/web.env" >/dev/null
    echo "wrote the web settings file"
fi
sudo ls -l "$BASE/shared/api.env" "$BASE/shared/web.env"
echo "API settings names: $(sudo cut -d= -f1 "$BASE/shared/api.env" | tr '\n' ' ')"
echo "web settings names: $(sudo cut -d= -f1 "$BASE/shared/web.env" | tr '\n' ' ')"

step "systemd units"
for unit in oriel-vue-web@.service oriel-vue-api@.service; do
    sudo install -o root -g root -m 644 "$HERE/$unit" "/etc/systemd/system/$unit"
    echo "installed $unit"
done
sudo systemctl daemon-reload

step "Activate script"
sudo install -o root -g root -m 755 "$HERE/activate.sh" /usr/local/bin/oriel-vue-activate
ls -l /usr/local/bin/oriel-vue-activate

step "Deploy keys"
touch ~/.ssh/authorized_keys
backup=~/.ssh/authorized_keys.bak-$(date -u +%F)
if [[ ! -e "$backup" ]]; then
    cp -p ~/.ssh/authorized_keys "$backup"
    echo "backed up authorized_keys to $backup"
fi
while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    key="$(awk '{print $(NF-1)}' <<<"$line")"
    if grep -qF "$key" ~/.ssh/authorized_keys; then
        echo "already present: $(awk '{print $NF}' <<<"$line")"
    else
        printf '%s\n' "$line" >> ~/.ssh/authorized_keys
        echo "added: $(awk '{print $NF}' <<<"$line")"
    fi
done < "$HERE/deploy-keys.pub"
echo "authorized_keys lines: $(grep -c . ~/.ssh/authorized_keys)"
awk '/oriel-vue-(upload|activate)$/ {print "line " NR ": " $NF}' ~/.ssh/authorized_keys
kept_lines="$(grep -c . "$backup")"
if cmp -s <(head -n "$kept_lines" ~/.ssh/authorized_keys) "$backup"; then
    echo "lines 1 to $kept_lines match the backup"
else
    echo "lines 1 to $kept_lines do NOT match the backup" >&2
fi

step "nginx"
echo "nginx -t before:"
if ! sudo nginx -t 2>&1 | awk '{print "  " $0}'; then
    echo "nginx does not pass its own test before any change. Stopping without touching it." >&2
    exit 70
fi
sudo install -d -o root -g root -m 755 /etc/nginx/oriel-vue
# Until the first release, the blue slot is named live. Nothing listens there
# yet, so visitors get the unavailable page.
if [[ ! -f /etc/nginx/oriel-vue/web.conf ]]; then
    echo "server 127.0.0.1:8700;" | sudo tee /etc/nginx/oriel-vue/web.conf >/dev/null
fi
if [[ ! -f /etc/nginx/oriel-vue/api.conf ]]; then
    echo "server 127.0.0.1:8702;" | sudo tee /etc/nginx/oriel-vue/api.conf >/dev/null
fi
sudo install -o root -g root -m 644 "$HERE/nginx.conf" /etc/nginx/sites-available/oriel-vue.jeffgicharu.com
sudo ln -sfn /etc/nginx/sites-available/oriel-vue.jeffgicharu.com /etc/nginx/sites-enabled/oriel-vue.jeffgicharu.com
echo "nginx -t after:"
if sudo nginx -t 2>&1 | awk '{print "  " $0}'; then
    sudo systemctl reload nginx
    echo "nginx reloaded with oriel-vue.jeffgicharu.com"
else
    sudo rm -f /etc/nginx/sites-enabled/oriel-vue.jeffgicharu.com
    sudo nginx -t || true
    echo "The oriel-vue nginx file failed the test and was taken out again." >&2
    exit 1
fi

step "Checks"
command -v rrsync >/dev/null && echo "rrsync: $(command -v rrsync)"
curl -s -o /dev/null -w 'oriel-vue.jeffgicharu.com through nginx before the first release: %{http_code}\n' --insecure --resolve oriel-vue.jeffgicharu.com:443:127.0.0.1 https://oriel-vue.jeffgicharu.com/ || true
free -m | head -2
df -h /var/www | tail -1

echo
echo "Provisioning finished."

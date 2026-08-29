#!/bin/sh

set -eu

REPO_URL="${SCRIPTABLES_REPO:-https://github.com/kevincoder-co-za/scriptables.git}"
APP_DIR="${SCRIPTABLES_DIR:-scriptables}"
APP_PORT="${SCRIPTABLES_PORT:-3012}"

log() { echo "==> $*"; }
die() { echo "Error: $*" >&2; exit 1; }

random_secret() {
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$1"
}

command -v git >/dev/null || die "git is required. Install it and re-run."
command -v curl >/dev/null || die "curl is required. Install it and re-run."

if [ -d "$APP_DIR/.git" ]; then
    log "Updating existing checkout in $APP_DIR"
    cd "$APP_DIR"
    git pull --ff-only || log "Could not fast forward, continuing with the current checkout."
elif [ -d "$APP_DIR" ]; then
    cd "$APP_DIR"
elif [ -f "docker-compose.yml" ] && [ -d "scriptables" ]; then
    log "Already inside a Scriptables checkout"
else
    log "Cloning $REPO_URL into $APP_DIR"
    git clone "$REPO_URL" "$APP_DIR"
    cd "$APP_DIR"
fi

[ -f docker-compose.yml ] || die "docker-compose.yml not found in $(pwd). Run this from outside the repo, or inside a checkout."

if ! command -v docker >/dev/null; then
    log "Installing Docker"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm -f get-docker.sh

    if [ "$(id -u)" -eq 0 ]; then
        [ -n "${SUDO_USER:-}" ] && usermod -aG docker "$SUDO_USER" || true
    else
        sudo usermod -aG docker "$(id -un)" || true
    fi

    log "Docker installed. You may need to log out and back in for group membership to apply."
fi

if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
elif command -v docker-compose >/dev/null; then
    COMPOSE="docker-compose"
else
    die "Neither 'docker compose' nor 'docker-compose' is available."
fi

SSH_DIR="${HOME:-/root}/.ssh"
if [ ! -d "$SSH_DIR" ]; then
    log "Creating $SSH_DIR"
    mkdir -p "$SSH_DIR"
    chmod 700 "$SSH_DIR"
fi

if [ -z "$(find "$SSH_DIR" -maxdepth 1 -name 'id_*' ! -name '*.pub' -print -quit 2>/dev/null)" ]; then
    log "No SSH key found in $SSH_DIR. Create one with: ssh-keygen -t ed25519"
fi

if [ -f .env ]; then
    log "Keeping the existing .env file"
else
    log "Writing .env with freshly generated secrets"

    USER_TIMEZONE="UTC"
    if [ -f /etc/timezone ]; then
        USER_TIMEZONE=$(cat /etc/timezone)
    elif [ -L /etc/localtime ]; then
        USER_TIMEZONE=$(readlink /etc/localtime | sed 's|.*/zoneinfo/||')
    fi

    MYSQL_USER_PASSWORD=$(random_secret 24)
    MYSQL_ROOT_PASSWORD=$(random_secret 24)
    ENCRYPTION_KEY=$(random_secret 32)

    cat > .env <<EOF
MYSQL_HOST=scriptables-db
MYSQL_PORT=3306
MYSQL_DATABASE=scriptables
MYSQL_USER=scriptable
MYSQL_PASSWORD=$MYSQL_USER_PASSWORD
MYSQL_ROOT_PASSWORD=$MYSQL_ROOT_PASSWORD

REDIS_DSN=scriptables-redis:6379

SCRIPTABLES_SERVER_DSN_HOST=0.0.0.0
SCRIPTABLES_SERVER_DSN_PORT=$APP_PORT
SCRIPTABLE_URL=http://127.0.0.1:$APP_PORT
ALLOWED_IPS=127.0.0.1

ENCRYPTION_KEY=$ENCRYPTION_KEY
ALLOW_REGISTER=true

SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_PORT=586
SMTP_USERNAME=xxxx
SMTP_PASSWORD=xxxx
SMTP_FROM_EMAIL=Scriptables <noreply@test.com>

TZ=$USER_TIMEZONE
VERBOSE_LOG=yes
GIN_MODE=release
EOF

    chmod 600 .env
fi

log "Building and starting the stack"
$COMPOSE -f docker-compose.yml up -d --build

log "Install complete. Visit http://127.0.0.1:$APP_PORT/users/register to create your admin account."
log "Once registered, set ALLOW_REGISTER=false in .env and run: $COMPOSE restart"

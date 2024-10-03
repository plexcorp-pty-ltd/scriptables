#!/bin/bash

if [ ! -d "./scriptables" ]; then
  git clone https://github.com/plexcorp-pty-ltd/scriptables.git scriptables
  cd scriptables
fi


if ! command -v docker >/dev/null; then
    echo "Failed to execute docker. Please install docker first before attempting to run this script. Lear more here: https://docs.docker.com/engine/install/ubuntu/"
    exit 0
fi

if ! command -v docker-compose >/dev/null; then
    echo "Failed to execute docker-compose. Please install docker-compose first before attempting to run this script. Lear more here: https://docs.docker.com/compose/install/linux/"
    exit 0
fi


USER_TIMEZONE=$(cat /etc/timezone)
MYSQL_USER_PASSWORD=$(date +%s | sha256sum | base64 | head -c 15)
MYSQL_ROOT_PASSWORD=$(date +%s | sha256sum | base64 | head -c 15)
ENCRYPTION_KEY=$(date +%s | sha256sum | base64 | head -c 16)

echo "
MYSQL_HOST=scriptables-db
MYSQL_PORT=3306
MYSQL_DATABASE=scriptables
MYSQL_USER=root
MYSQL_PASSWORD=1234
MYSQL_ROOT_PASSWORD=1234
SCRIPTABLE_URL=http://127.0.0.1:3012
ALLOWED_IPS=127.0.0.1
REDIS_DSN=127.0.0.1:6379
ENCRYPTION_KEY=$ENCRYPTION_KEY
SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_USERNAME=xxxx
SMTP_PASSWORD=xxx
SMTP_PORT=586
ALLOW_REGISTER=true
SMTP_FROM_EMAIL=Scriptables <noreply@test.com>
TZ=$USER_TIMEZONE
SCRIPTABLES_SERVER_DSN_HOST=0.0.0.0
SCRIPTABLES_SERVER_DSN_PORT=3012
VERBOSE_LOG=yes
GIN_MODE=release
" > .env

docker-compose -f docker-compose.yml up -d --build

echo "Install complete. Please visit: http://127.0.0.1:3012/users/register"

export DEBIAN_FRONTEND=noninteractive

sudo apt-get update -y
sudo apt-get install -y ufw fail2ban

if grep -qE '^[[:space:]]*#?[[:space:]]*Port[[:space:]]+[0-9]+' /etc/ssh/sshd_config; then
    sudo sed -i -E 's/^[[:space:]]*#?[[:space:]]*Port[[:space:]]+[0-9]+.*/Port #NEW_SSH_PORT#/' /etc/ssh/sshd_config
else
    echo "Port #NEW_SSH_PORT#" | sudo tee -a /etc/ssh/sshd_config
fi

if ! grep -qE "^AllowUsers[[:space:]]+#username#$" /etc/ssh/sshd_config; then
    sudo sh -c "echo 'AllowUsers #username#' >> /etc/ssh/sshd_config"
fi

sudo ufw default deny incoming
sudo ufw default allow outgoing

sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow #NEW_SSH_PORT#/tcp
sudo ufw allow 53

sudo ufw --force enable
sudo ufw status numbered

if [ ! -f /etc/fail2ban/jail.local ]; then
    sudo cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local
fi

sudo sed -i 's/# ignoreip = 127.0.0.1/ignoreip = 127.0.0.1/; s/# bantime = 10m/bantime = 1h/; s/# findtime = 10m/findtime = 10m/; s/# maxretry = 5/maxretry = 3/' /etc/fail2ban/jail.local

sudo tee /etc/fail2ban/jail.d/scriptables-sshd.conf >/dev/null <<EOF
[sshd]
enabled = true
port    = #NEW_SSH_PORT#
backend = systemd
maxretry = 3
bantime  = 1h
findtime = 10m
EOF

sudo systemctl enable fail2ban
sudo systemctl restart fail2ban
sudo fail2ban-client status sshd || true

sudo sshd -t && sudo systemctl restart ssh

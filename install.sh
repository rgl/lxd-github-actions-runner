#!/bin/bash
set -euo pipefail

if [ ! -v GITHUB_TOKEN ]; then
    echo "ERROR: The GITHUB_TOKEN environment variable is not set"
    exit 1
fi

set -x

if [ -z "$(id -g lxd-ghar 2>/dev/null)" ]; then
    groupadd --system lxd-ghar
fi

if [ -z "$(id -u lxd-ghar 2>/dev/null)" ]; then
    # NB default apparmor settings only let us use lxc in /home, thats why
    #    we do not use /src/lxd-ghar home.
    adduser \
        --system \
        --disabled-login \
        --no-create-home \
        --home /home/lxd-ghar \
        --gecos 'LXD GitHub Actions Runner' \
        --ingroup lxd-ghar \
        lxd-ghar
    usermod -aG lxd lxd-ghar
fi

install -d -o lxd-ghar -g lxd-ghar -m 750 /home/lxd-ghar
install -d -o root     -g lxd-ghar -m 750 /etc/lxd-ghar
install    -o root     -g root     -m 755 lxd-github-actions-runner /usr/local/bin
install    -o root     -g lxd-ghar -m 640 config.yml /etc/lxd-ghar/config.yml
install    -o root     -g root     -m 600 /dev/null /etc/lxd-ghar/environment

cat >/etc/lxd-ghar/environment <<EOF
GITHUB_TOKEN=$GITHUB_TOKEN
EOF

cat >/etc/systemd/system/lxd-ghar.service <<'EOF'
[Unit]
Description=LXD GitHub Actions Runner
Requires=network-online.target
After=network-online.target

[Service]
Type=simple
User=lxd-ghar
Group=lxd-ghar
WorkingDirectory=/home/lxd-ghar
EnvironmentFile=/etc/lxd-ghar/environment
ExecStart=/usr/local/bin/lxd-github-actions-runner
Restart=always
#RestartSecs=15

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable lxd-ghar
systemctl start lxd-ghar

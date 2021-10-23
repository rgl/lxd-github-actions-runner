#!/bin/bash
set -euxo pipefail

if [ -f /etc/systemd/system/lxd-ghar.service ]; then
    systemctl disable lxd-ghar --now
    rm -f /etc/systemd/system/lxd-ghar.service
fi

rm -rf /usr/local/bin/lxd-github-actions-runner
rm -rf /etc/lxd-ghar
rm -rf /home/lxd-ghar

if [ -n "$(id -u lxd-ghar 2>/dev/null)" ]; then
    deluser lxd-ghar
fi

if [ -n "$(id -g lxd-ghar 2>/dev/null)" ]; then
    groupdel lxd-ghar
fi

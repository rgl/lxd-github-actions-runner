#!/bin/bash
set -euxo pipefail

image_name='ghar-debian-11'
base_image_name='images:debian/11'

if [ -n "$(lxc list --format csv $image_name)" ]; then
    lxc delete --force $image_name
fi

lxc launch $base_image_name $image_name

lxc exec $image_name -- bash <<'LXCEXEC'
set -euxo pipefail

# wait for the system to be fully running.
while [ "$(systemctl is-system-running)" != "running" ]; do sleep 1; done

# configure the shell.
cat >/etc/vim/vimrc.local <<'EOF'
syntax on
set background=dark
set esckeys
set ruler
set laststatus=2
set nobackup
EOF
cat >/etc/profile.d/local.sh <<'EOF'
export EDITOR=vim
export PAGER=less
alias l='ls -lF --color'
alias ll='l -a'
alias h='history 25'
alias j='jobs -l'
EOF
cat >/etc/inputrc <<'EOF'
set input-meta on
set output-meta on
set show-all-if-ambiguous on
set completion-ignore-case on
"\e[A": history-search-backward
"\e[B": history-search-forward
"\eOD": backward-word
"\eOC": forward-word
EOF

# add the GitHub Actions Runner user.
groupadd ghar
adduser \
    --disabled-login \
    --no-create-home \
    --home /home/ghar \
    --gecos 'GitHub Actions Runner' \
    --ingroup ghar \
    ghar
install -d -o ghar -g ghar -m 750 /home/ghar

# install dependencies.
apt-get install -y git curl

# install the GitHub Actions Runner.
# see https://github.com/actions/runner/releases
ghar_version='2.283.3'
if [ "$(uname -m)" == "aarch64" ]; then
    ghar_arch=arm64
else
    ghar_arch=x64
fi
ghar_url="https://github.com/actions/runner/releases/download/v$ghar_version/actions-runner-linux-$ghar_arch-$ghar_version.tar.gz"
curl -s -L -o /tmp/ghar.tar.gz $ghar_url
install -d -o ghar -g ghar /home/ghar/runner
tar xf /tmp/ghar.tar.gz --no-same-owner -C /home/ghar/runner
chown -R ghar:ghar /home/ghar/runner
rm -f /tmp/ghar.tar.gz

# install dependencies.
/home/ghar/runner/bin/installdependencies.sh
LXCEXEC

lxc stop $image_name
lxc config set $image_name boot.autostart=false
lxc config set $image_name security.idmap.isolated=true

# About

[![build](https://github.com/rgl/lxd-github-actions-runner/actions/workflows/build.yml/badge.svg)](https://github.com/rgl/lxd-github-actions-runner/actions/workflows/build.yml)

Execute a self-hosted GitHub Actions Runner in a ephemeral LXD container.

# Usage

Build the `ghar-debian-11` runner container:

```bash
./images/ghar-debian-11/build.sh
```

Create the `lxd-ghar` Personal Access Token (PAT) at
https://github.com/settings/tokens/new for the `repo` scope.

Create the configuration file:

```bash
cp example-config.yml config.yml
vim config.yml
```

Build and execute the runner:

```bash
go build
export LXD_SOCKET='/var/snap/lxd/common/lxd/unix.socket'
export GITHUB_TOKEN='your-lxd-ghar-pat'
./lxd-github-actions-runner -config config.yml
```

Install and start the `lxd-ghar` service:

```bash
sudo --preserve-env=GITHUB_TOKEN ./install.sh
```

Later you can uninstall with:

```bash
sudo ./uninstall.sh
```

# Reference

* GitHub
  * [About self-hosted runners](https://docs.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
  * [Using self-hosted runners in a workflow](https://docs.github.com/en/actions/hosting-your-own-runners/using-self-hosted-runners-in-a-workflow)
  * [Self-hosted runners API](https://docs.github.com/en/rest/reference/actions#self-hosted-runners)
  * [Documentation](https://github.com/actions/runner/tree/main/docs)
  * [Source code repository](https://github.com/actions/runner)
* LXD/LXC
  * [Documentation](https://github.com/lxc/lxd/tree/master/doc)
    * [Instance configuration](https://github.com/lxc/lxd/blob/master/doc/instances.md)
  * [lxd-github-actions](https://github.com/stgraber/lxd-github-actions)

# PPA publishing

See **[`ppa.md`](../ppa.md)** at the repository root for the full Launchpad PPA guide (GPG, build, upload, troubleshooting).

Quick publish:

```bash
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

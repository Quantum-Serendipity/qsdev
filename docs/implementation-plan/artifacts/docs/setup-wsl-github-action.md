# Setup WSL GitHub Action
> Source: https://github.com/marketplace/actions/setup-wsl
> Retrieved: 2026-05-12

## Supported Distributions

- Alpine: 3.17-3.23
- Debian: 11-13 (Debian-13 default)
- Ubuntu: 16.04-24.04
- kali-linux, openSUSE-Leap-15.2

## WSL Versions

- WSLv1: windows-2019 runners
- WSLv2: windows-2022 and later (default)

## Running Commands in WSL

```yaml
- shell: wsl-bash {0}
  run: id
```

Distribution-specific wrappers: `wsl-bash_Ubuntu-20.04`

## Key Notes

- wsl --install works on windows-2025 runner
- Automatic installation of specified distributions
- Can write custom /etc/wsl.conf
- Package management (update, install packages)

<!-- Sources: https://goreleaser.com/customization/homebrew/ https://goreleaser.com/customization/publish/scoop/ https://goreleaser.com/customization/chocolatey/ https://goreleaser.com/customization/winget/ -->
<!-- Retrieved: 2026-05-12 -->

# GoReleaser Package Manager Configurations

## Homebrew Casks

```yaml
homebrew_casks:
  - name: myapp
    binaries:
      - myapp
    repository:
      owner: myorg
      name: homebrew-tap
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    dependencies:
      - formula: git
    hooks:
      post_install: |
        system "#{bin}/myapp", "setup"
```

## Scoop Manifests

```yaml
scoops:
  - name: myapp
    homepage: https://example.com
    description: My CLI tool
    license: MIT
    repository:
      owner: myorg
      name: scoop-bucket
      branch: main
      token: "{{ .Env.SCOOP_TAP_GITHUB_TOKEN }}"
```

## Chocolatey Packages

```yaml
chocolateys:
  - name: myapp
    title: My App
    owners: myorg
    authors: My Org
    project_url: https://example.com
    license_url: https://github.com/myorg/myapp/blob/main/LICENSE
    description: My CLI tool
    api_key: "{{ .Env.CHOCOLATEY_API_KEY }}"
    source_repo: "https://push.chocolatey.org/"
```

## Winget Manifests

```yaml
winget:
  - name: myapp
    publisher: MyOrg
    short_description: My CLI tool
    license: MIT
    publisher_url: https://example.com
    publisher_support_url: https://github.com/myorg/myapp/issues
    package_identifier: MyOrg.MyApp
    repository:
      owner: microsoft
      name: winget-pkgs
      branch: main
      token: "{{ .Env.WINGET_GITHUB_TOKEN }}"
    pull_request:
      enabled: true
```

Notes:
- Winget requires PR to microsoft/winget-pkgs (manual review)
- Chocolatey packages are manually reviewed
- Scoop and Homebrew can use your own tap/bucket repos

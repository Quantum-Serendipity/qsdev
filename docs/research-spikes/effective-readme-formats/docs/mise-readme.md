<!-- Source: https://raw.githubusercontent.com/jdx/mise/main/README.md -->
<!-- Retrieved: 2026-05-15 -->

<div align="center">

<h1 align="center">
  <a href="https://mise.en.dev">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="docs/public/logo-dark.svg" />
      <img src="docs/public/logo-light.svg" alt="mise" width="256" height="256" />
    </picture>
    <br>
    mise-en-place
  </a>
</h1>

<p>
  <a href="https://crates.io/crates/mise"><img alt="Crates.io" src="https://img.shields.io/crates/v/mise?style=for-the-badge&color=8B2252"></a>
  <a href="https://github.com/jdx/mise/blob/main/LICENSE"><img alt="GitHub" src="https://img.shields.io/github/license/jdx/mise?style=for-the-badge&color=6B7F4E"></a>
  <a href="https://github.com/jdx/mise/actions/workflows/test.yml"><img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/jdx/mise/test.yml?style=for-the-badge&color=C5975B"></a>
  <a href="https://discord.gg/mABnUDvP57"><img alt="Discord" src="https://img.shields.io/discord/1066429325269794907?style=for-the-badge&color=8B2252"></a>
</p>

<p><b>Dev tools, env vars, and tasks in one CLI</b></p>

<p align="center">
  <a href="https://mise.en.dev/getting-started.html">Getting Started</a> |
  <a href="https://mise.en.dev">Documentation</a> |
  <a href="https://mise.en.dev/dev-tools/">Dev Tools</a> |
  <a href="https://mise.en.dev/environments/">Environments</a> |
  <a href="https://mise.en.dev/tasks/">Tasks</a>
</p>

<hr />

</div>

> [!TIP]
> My latest project, [aube](https://aube.en.dev) just hit stable! It's the fastest Node.js package manager with strong security defaults and is compatible with npm/pnpm/yarn lockfiles!

## What is it?

`mise` prepares your development environment before each command runs. It keeps project tools, environment variables, and tasks in one `mise.toml` file so new shells, checkouts, and CI jobs all start from the same setup.

- Install and switch between [dev tools](https://mise.en.dev/dev-tools/) like node, python, cmake, terraform, and [hundreds more](https://mise.en.dev/registry.html).
- Load [environment variables](https://mise.en.dev/environments/) per project directory, including values from `.env` files and other sources.
- Define and run [tasks](https://mise.en.dev/tasks/) for building, testing, linting, and deploying projects.

## Demo

The following demo shows how to install and use `mise` to manage multiple versions of `node` on the same system. Note that calling `which node` gives us a real path to node, not a shim.

[![demo](./docs/tapes/demo.gif)](https://mise.en.dev/demo.html)

See [demo transcript](https://mise.en.dev/demo.html).

## Quickstart

### Install mise

```sh-session
$ curl https://mise.run | sh
$ ~/.local/bin/mise --version
mise-en-place 2026.5.9 macos-arm64 (2026-05-15)
```

Hook mise into your shell (pick the right one for your shell):

```sh-session
echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc
echo 'eval "$(~/.local/bin/mise activate zsh)"' >> ~/.zshrc
echo '~/.local/bin/mise activate fish | source' >> ~/.config/fish/config.fish
echo '~/.local/bin/mise activate pwsh | Out-String | Invoke-Expression' >> ~/.config/powershell/Microsoft.PowerShell_profile.ps1
```

### Execute commands with specific tools

```sh-session
$ mise exec node@26 -- node -v
mise node@26.x.x installed
v26.x.x
```

### Install tools

```sh-session
$ mise use --global node@26 go@1
$ node -v
v26.x.x
$ go version
go version go1.x.x macos/arm64
```

### Manage environment variables

```toml
# mise.toml
[env]
SOME_VAR = "foo"
```

```sh-session
$ mise set SOME_VAR=bar
$ echo $SOME_VAR
bar
```

### Run tasks

```toml
# mise.toml
[tasks.build]
description = "build the project"
run = "echo building..."
```

```sh-session
$ mise run build
building...
```

### Example mise project

```toml
# mise.toml
[tools]
terraform = "1"
aws-cli = "2"

[env]
TF_WORKSPACE = "development"
AWS_REGION = "us-west-2"
AWS_PROFILE = "dev"

[tasks.plan]
description = "Run terraform plan with configured workspace"
run = """
terraform init
terraform workspace select $TF_WORKSPACE
terraform plan
"""

[tasks.validate]
description = "Validate AWS credentials and terraform config"
run = """
aws sts get-caller-identity
terraform validate
"""

[tasks.deploy]
description = "Deploy infrastructure after validation"
depends = ["validate", "plan"]
run = "terraform apply -auto-approve"
```

Run it with:

```sh-session
mise install # install tools specified in mise.toml
mise run deploy
```

## Full Documentation

See [mise.en.dev](https://mise.en.dev)

## GitHub Issues & Discussions

Due to the volume of issue submissions mise received, using GitHub Issues became unsustainable for the project. Instead, mise uses GitHub Discussions.

## Special Thanks

Thanks to [Namespace](https://namespace.so) for providing CI services for mise.

## Contributors

[![Contributors](https://contrib.rocks/image?repo=jdx/mise)](https://github.com/jdx/mise/graphs/contributors)

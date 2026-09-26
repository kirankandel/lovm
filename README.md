# lovm

Install several LibreOffice versions side by side and switch between them, like
[nvm](https://github.com/nvm-sh/nvm) does for Node.

No root or admin rights needed. Everything lives in `~/.lovm`.

## Status

| | |
|---|---|
| Platforms | Linux and macOS, x86_64 and arm64. Windows is planned. |
| Versions | Official releases from 6.0 onwards, from the [LibreOffice archive](https://downloadarchive.documentfoundation.org/libreoffice/old/) |
| Apple Silicon | Versions before 7.2 have no native build; the x86_64 build runs under Rosetta 2 |

## Install

Requires Go 1.22+.

```sh
git clone <this repo> && cd lovm
go install .
```

Then put the shim directory on your `PATH`, for example in `~/.zshrc` or `~/.bashrc`:

```sh
export PATH="$HOME/.lovm/bin:$PATH"
```

`lovm install` creates `~/.lovm/bin/soffice` the first time you install a version.

## Quick start

```sh
lovm install 24.8          # newest 24.8.x release for this machine
lovm install 7.6
lovm default 24.8          # used everywhere unless something overrides it

cd ~/projects/customer-repro
lovm use 7.6               # writes .lovmrc here
soffice --version          # LibreOffice 7.6.x

lovm exec 24.8 -- soffice --headless --convert-to pdf report.docx
```

## Commands

| Command | What it does |
|---|---|
| `lovm ls-remote [prefix] [--all] [--refresh]` | Versions you can install on this machine. `--all` also shows RCs and builds missing for your platform |
| `lovm install <version>` | Download, unpack and verify a version |
| `lovm uninstall <version>` | Remove an installed version (the spec must match exactly one) |
| `lovm ls` | Installed versions; `*` marks the active one and where it was chosen |
| `lovm use <version>` | Pin a version for this directory (`.lovmrc`) |
| `lovm default <version>` | Set the global default |
| `lovm current` | The active version and why it was chosen |
| `lovm which [version]` | Path of the real `soffice` binary |
| `lovm exec <version> -- <cmd...>` | Run a command with that version active |
| `lovm cache clear` | Delete downloaded installers and cached version lists |

### Version specs

| You type | You get |
|---|---|
| `latest` | Newest release available for your platform |
| `24.8` | Newest `24.8.x` release available for your platform |
| `24.8.4` | The 24.8.4 release |
| `24.8.4.2` | Exactly that build, even a release candidate |

Release candidates never match partial specs or `latest`, so `lovm install 26.8` can't
give you an RC by accident. To test an RC, name its full build number
(`lovm ls-remote 26.8 --all` lists them).

## How the active version is chosen

When anything runs `soffice`, the shim picks the version from the first of these that is set:

1. the `LOVM_VERSION` environment variable
2. the nearest `.lovmrc`, searching from the current directory upwards
3. `~/.lovm/default`

It then runs the newest installed build matching that spec. If nothing is selected, or the
selected version isn't installed, it fails with a clear message. **It never falls back to
a system LibreOffice**, so you can't reproduce a bug on the wrong version by accident.

`.lovmrc` holds one line such as `24.8` (follows new 24.8.x installs) or `24.8.4.2`
(exact).

## Separate profiles

Each version gets its own LibreOffice user profile in `~/.lovm/versions/<build>/profile`.
Normally LibreOffice runs one instance per profile and hands new work to the instance
that is already running, so two versions sharing a profile would quietly run both
conversions in whichever started first. Separate profiles avoid that and let different
versions run at the same time.

If you pass your own `-env:UserInstallation=...`, lovm leaves it alone.

## Errors and exit codes

Every failure says what went wrong and what to do next, and exits with its own code so
scripts can tell them apart:

| Code | Meaning |
|---|---|
| 2 | Wrong command usage |
| 3 | Version not found in the archive (the closest versions are suggested) |
| 4 | Version older than 6.0 |
| 5 | No build for your OS/architecture (shows which platforms have one, and the first version that supports yours) |
| 6 | Needs Rosetta 2, which isn't installed |
| 7 | Only release candidates match |
| 8 | Operating system not supported yet |
| 9 | Unpacked, but LibreOffice won't start on this system (its output is shown) |
| 10 | No version selected |
| 11 | Selected version not installed |

`lovm exec` exits with the command's own exit code.

## Where things live

```
~/.lovm/                      override with LOVM_HOME
  bin/soffice                 the shim (a symlink to lovm)
  default                     global default spec
  versions/<build>/
    install/                  the unpacked LibreOffice
    profile/                  this version's user profile
    meta.json                 where it came from and when
  cache/
    catalog.json              archive build list (refreshed daily)
    installers.json           which builds exist for which platform (kept forever)
    downloads/                installers, so reinstalling doesn't download again
```

Installs are atomic: a version only appears once it has been unpacked and has passed a
`soffice --version` check. An interrupted install leaves nothing half-installed.

## Development

```sh
go test ./...
go vet ./...
```

The end-to-end test downloads real LibreOffice builds (~200 MB each), so it only runs
when you ask for it:

```sh
LOVM_INTEGRATION=1 go test -run Integration -v -timeout 30m .
LOVM_INTEGRATION=1 LOVM_INTEGRATION_VERSIONS="6.4 24.8" go test -run Integration -v -timeout 40m .
```

Design notes: [`docs/superpowers/specs/2026-09-24-lovm-design.md`](docs/superpowers/specs/2026-09-24-lovm-design.md).

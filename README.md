# lovm

Install several LibreOffice versions side by side and switch between them, like
[nvm](https://github.com/nvm-sh/nvm) does for Node.

No root or admin rights needed. Everything lives in `~/.lovm`.

## Status

|  |  |
| --- | --- |
| Platforms | Linux, macOS and Windows, x86_64 and arm64. Windows support is new and less tested. |
| Versions | Official releases from 6.0 onwards, from the [LibreOffice archive](https://downloadarchive.documentfoundation.org/libreoffice/old/) |
| Apple Silicon | Versions before 7.2 have no native build; the x86_64 build runs under Rosetta 2 |
| Windows on ARM | Only 7.4 and newer, the versions with an ARM build |

### Linux system libraries

LibreOffice's Linux builds use the system's shared libraries. Desktops usually have them
already; minimal servers and Docker images don't. For headless use on Debian/Ubuntu:

```sh
sudo apt-get install --no-install-recommends \
  libxml2 libxslt1.1 libglib2.0-0 libnss3 libcairo2 libfontconfig1 libfreetype6 \
  libcups2 libdbus-1-3 libsm6 libice6 libx11-6 libx11-xcb1 libxext6 libxinerama1 \
  libxrandr2 libxrender1 libxcb1 fonts-dejavu-core
```

If one is missing, `lovm install` stops and names it, for example
`error while loading shared libraries: libxml2.so.2`.

## Build and install

You need [Go 1.22 or newer](https://go.dev/dl/) (`go version` to check).

### 1. Build the `lovm` binary

```sh
git clone <this repo> lovm
cd lovm
go build -o lovm .        # produces ./lovm
./lovm help               # quick check that it runs
```

### 2. Put it somewhere on your `PATH`

Either let Go do it:

```sh
go install .              # installs to $(go env GOPATH)/bin, usually ~/go/bin
```

or copy the binary yourself:

```sh
mkdir -p ~/.local/bin
cp lovm ~/.local/bin/
```

Make sure that directory is on your `PATH`: `which lovm` should print it. If you get
`command not found`, add it. For `go install`:

```sh
export PATH="$HOME/go/bin:$PATH"   # add to ~/.zshrc or ~/.bashrc
```

### 3. Add lovm's shim directory to your `PATH`

This is what makes plain `soffice` use the version lovm selected. Add it **before** any
other directory that might contain a LibreOffice:

```sh
# zsh: ~/.zshrc    bash: ~/.bashrc
export PATH="$HOME/.lovm/bin:$PATH"
```

```fish
# fish: ~/.config/fish/config.fish
fish_add_path --prepend ~/.lovm/bin
```

Open a new terminal afterwards. The shim itself (`~/.lovm/bin/soffice`) is created by your
first `lovm install`.

### 4. Check it works

```sh
lovm install latest
lovm default latest
hash -r                   # zsh/bash: forget where soffice was before the shim existed
which soffice             # ~/.lovm/bin/soffice
soffice --version         # the version you just installed
```

If you also have a system LibreOffice (for example `/usr/local/bin/soffice`) and your
shell still runs it, it can crash with `DeploymentException`: when started by name, macOS
LibreOffice finds its own files by searching `PATH`, and now finds lovm's shim first. Run
`hash -r` or open a new terminal so `soffice` goes through the shim.

### Windows

In PowerShell:

```powershell
git clone <this repo> lovm
cd lovm
go install .                       # installs to %USERPROFILE%\go\bin, which the Go installer puts on PATH

# Add lovm's shim directory to your user PATH, then open a new terminal
[Environment]::SetEnvironmentVariable("Path", "$env:USERPROFILE\.lovm\bin;" + [Environment]::GetEnvironmentVariable("Path", "User"), "User")

lovm install latest
lovm default latest
where.exe soffice                  # %USERPROFILE%\.lovm\bin\soffice.exe first
soffice --version
```

Differences from Linux and macOS:

- Installers are unpacked with `msiexec /a` (an "administrative install"): the files are
  copied out without touching the registry, the Start menu or file associations, so it
  needs no admin rights and doesn't interfere with a regular LibreOffice install.
- The shim `~\.lovm\bin\soffice.exe` is a **copy** of `lovm.exe`, because symlinks need
  admin rights on Windows. `lovm install` refreshes it; after updating lovm, run any
  `lovm install` (e.g. of a version you already have) to update the shim.
- The shim runs LibreOffice's `soffice.com`, the console launcher, so `--version` output and
  conversion errors show up in the terminal (`soffice.exe` is a GUI program and prints
  nothing).

### Updating and removing lovm

- **Update:** `git pull && go install .` The shim is a symlink to the `lovm` binary, so
  it picks up the new version automatically. If you moved the binary, run any
  `lovm install` again to re-point the shim.
- **Remove:** delete the `lovm` binary and `~/.lovm` (this also deletes every installed
  LibreOffice), then remove the `PATH` line from your shell config.

## Using it from a service, script or CI

Services (systemd, launchd, Docker entrypoints, cron) don't read your shell config, so
the `PATH` line from step 3 doesn't apply to them. Either:

- call the shim by its full path, `~/.lovm/bin/soffice`, wherever the service is
  configured with the LibreOffice binary; or
- set `PATH=$HOME/.lovm/bin:$PATH` in the service's environment.

To choose the version, set `LOVM_VERSION` in the service environment (for example
`LOVM_VERSION=24.8`) or put a `.lovmrc` in its working directory. If neither is set, the
global default is used.

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
| --- | --- |
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
| --- | --- |
| `latest` | Newest release available for your platform |
| `fresh` | Newest release of TDF's **fresh** branch (newest features) |
| `still` | Newest release of TDF's **still** branch (the more mature one TDF recommends for conservative users) |
| `24.8` | Newest `24.8.x` release available for your platform |
| `24.8.4` | The 24.8.4 release |
| `24.8.4.2` | Exactly that build, even a release candidate |

Release candidates never match partial specs or `latest`, so `lovm install 26.8` can't
give you an RC by accident. To test an RC, name its full build number
(`lovm ls-remote 26.8 --all` lists them).

`ls-remote` marks the current fresh and still releases. They are the two branches
libreoffice.org offers for download; LibreOffice itself has no LTS releases.

## How the active version is chosen

When anything runs `soffice`, the shim picks the version from the first of these that is set:

1. the `LOVM_VERSION` environment variable
2. the nearest `.lovmrc`, searching from the current directory upwards
3. `~/.lovm/default`

It then runs the newest installed build matching that spec. If nothing is selected, or the
selected version isn't installed, it fails with a clear message. **It never falls back to
a system LibreOffice**, so you can't reproduce a bug on the wrong version by accident.

`.lovmrc` holds one line such as `24.8` (follows new 24.8.x installs), `24.8.4.2`
(exact) or `still` (follows TDF's still branch as of the last time lovm fetched the
version list; any `install` or `ls-remote` refreshes it).

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
| --- | --- |
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
| 12 | `fresh`/`still` can't be mapped to a branch yet (run `lovm ls-remote --refresh`) |

`lovm exec` exits with the command's own exit code.

## Where things live

```text
~/.lovm/                      override with LOVM_HOME
  bin/soffice                 the shim (a symlink to lovm; soffice.exe, a copy, on Windows)
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

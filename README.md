# Ashen Sync

Open-source cloud saves for **DARK SOULS II** and **DARK SOULS III** on Steam — **Windows, Linux, and Steam Deck**.

macOS is a fine place to **develop and cross-compile**, but not a runtime target: DS2/DS3 do not ship native Mac Steam builds, so save sync belongs on the machines where you actually play.

FromSoftware never shipped Steam Cloud for these titles. Ashen Sync fills that gap **without** embedding Dropbox/Google API secrets and **without** running a sync server that sees your saves.

## Security model

- **Your cloud, your credentials.** Sync uses a rclone remote *you* configure (`rclone config`). Ashen Sync only shells out to `rclone`.
- **No embedded OAuth client secrets** in this repo or binary.
- **No central Ashen Sync server.** We never receive your save files.
- **No silent overwrites.** Divergent local/remote saves produce a conflict; you choose local or remote. Local snapshots are kept before replace.
- **No save-folder symlinks into cloud storage.** That pattern has corrupted Souls saves when the provider locks files mid-write.
- **No telemetry.**

Threat note: whoever can use your rclone remote can read/write the dedicated folder you pointed at. Use a folder that exists only for Ashen Sync.

## How it works

1. **`play`** — pull → launch via `steam://rungameid/...` → wait for exit → push  
2. **`watch`** — background watcher with debounce; pushes only after a quiet period **and** only when the game process is not running; idle pulls on an interval  
3. **`gui`** — Fyne desktop UI for status, pull/push/play, watcher, and conflicts  

## Install

### Requirements

- [rclone](https://rclone.org/install/) on your `PATH` (or set `rclone_path` in config)
- Steam + DS2 SotFS and/or DS3
- Go 1.22+ to build from source

### Build (on a gaming PC / Deck)

```bash
git clone https://github.com/migzeetigglez/ashen-sync.git
cd ashen-sync
go build -o ashen-sync ./cmd/ashen-sync
```

GUI builds need CGO + OpenGL libs on that OS.

### Cross-compile CLI from a Mac (no GUI)

Fyne needs per-OS native builds. For Deck/Linux/Windows CLI + watcher from macOS:

```bash
# Linux / Steam Deck
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags nogui -o dist/ashen-sync-linux-amd64 ./cmd/ashen-sync

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags nogui -o dist/ashen-sync-windows-amd64.exe ./cmd/ashen-sync
```

Copy the binary + install [rclone](https://rclone.org/install/) on the play machine, then run `ashen-sync init` there. CI also builds platform artifacts on push.

### One-time setup

```bash
rclone config          # create a remote; prefer a dedicated folder
./ashen-sync init      # pick remote name + path (default: ashen-sync)
./ashen-sync doctor
```

## Usage

```bash
./ashen-sync games
./ashen-sync status ds3
./ashen-sync pull ds3
./ashen-sync push ds3
./ashen-sync play ds3
./ashen-sync watch
./ashen-sync gui
```

Conflict resolution:

```bash
./ashen-sync pull ds3 --prefer remote
./ashen-sync push ds3 --prefer local
```

Games: `ds2` (Scholar of the First Sin), `ds3`, `ds2-vanilla`.

### Steam Deck

1. Desktop Mode: install rclone (static binary in `$HOME/bin` is fine).
2. `rclone config` + `ashen-sync init`.
3. For Game Mode, add a non-Steam shortcut or game launch option that runs:
   ` /path/to/ashen-sync play ds3 `
4. Optional: run `ashen-sync watch` from Desktop Mode / autostart when you want idle sync without the launch wrapper.

### Config

- Linux/macOS: `~/.config/ashen-sync/config.toml`
- Windows: `%APPDATA%\ashen-sync\config.toml`
- Snapshots: `~/.local/share/ashen-sync/snapshots/` (Windows: `%LOCALAPPDATA%\ashen-sync\snapshots\`)

```toml
rclone_path = "rclone"
remote = "gdrive"
remote_path = "ashen-sync"
snapshot_keep = 10
quiet_period_seconds = 15
pull_interval_seconds = 300
```

Override a save path if auto-detect fails:

```toml
[game_overrides]
ds3 = "/full/path/to/DarkSoulsIII/7656119..."
```

## Save locations

| Platform | DS2 SotFS / DS3 |
|---|---|
| Windows | `%APPDATA%\DarkSoulsII\<steamid>\` / `%APPDATA%\DarkSoulsIII\<steamid>\` |
| Linux / Deck (Proton) | `~/.steam/steam/steamapps/compatdata/<appid>/pfx/drive_c/users/steamuser/AppData/Roaming/...` |

AppIDs: SotFS `335300`, DS3 `374320`, vanilla DS2 `236430`.

## Why not Dropbox junctions?

Cloud clients often lock files while uploading. Dark Souls writes saves frequently; a lock mid-write can corrupt the slot. Ashen Sync copies through a staging area via rclone and never mounts the live save directory as a cloud folder.

## License

MIT — see [LICENSE](LICENSE).

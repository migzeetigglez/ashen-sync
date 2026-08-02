# Save Channeler

Open-source cloud saves for **DARK SOULS II** and **DARK SOULS III** on Steam — **Windows, Linux, and Steam Deck**.

CLI binary: **`save-channeler`**.

macOS is a fine place to **develop and cross-compile**, but not a runtime target: DS2/DS3 do not ship native Mac Steam builds, so save sync belongs on the machines where you actually play.

FromSoftware never shipped Steam Cloud for these titles. Save Channeler fills that gap **without** embedding Dropbox/Google API secrets and **without** running a sync server that sees your saves.

## Security model

- **Your cloud, your credentials.** Sync uses a rclone remote *you* configure (`rclone config`). Save Channeler only shells out to `rclone`.
- **No embedded OAuth client secrets** in this repo or binary.
- **No central sync server.** We never receive your save files.
- **No silent overwrites.** Divergent local/remote saves produce a conflict; you choose local or remote. Local snapshots are kept before replace.
- **No save-folder symlinks into cloud storage.** That pattern has corrupted Souls saves when the provider locks files mid-write.
- **No telemetry.**

Threat note: whoever can use your rclone remote can read/write the dedicated folder you pointed at. Use a folder that exists only for Save Channeler.

## How it works

1. **`play`** — pull → launch via `steam://rungameid/...` → wait for exit → push  
2. **`watch`** — background watcher with debounce; pushes only after a quiet period **and** only when the game process is not running; idle pulls on an interval  
3. **`gui`** — Fyne desktop UI for status, pull/push/play, watcher, and conflicts  

App icon (Channeler helm + beige fedora) lives in [`assets/icon.png`](assets/icon.png) and is embedded in GUI builds.

## Install

### Requirements

- [rclone](https://rclone.org/install/) on your `PATH` (or set `rclone_path` in config)
- Steam + DS2 SotFS and/or DS3
- Go 1.22+ to build from source

### Package for Linux / Steam Deck / Windows

From this repo (works when cross-compiling from a Mac too):

```bash
./scripts/package-linux.sh      # → dist/save-channeler-linux-amd64.tar.gz
./scripts/package-windows.sh    # → dist/save-channeler-windows-amd64.zip
./scripts/package-macos.sh      # → dist/Save Channeler.app (dev machine)
```

**Linux / Deck:** copy the tarball over, then:

```bash
tar xzf save-channeler-linux-amd64.tar.gz
cd save-channeler-linux-amd64
./install.sh                    # installs to ~/.local/bin + .desktop launcher
```

**Windows:** extract the zip, install [rclone](https://rclone.org/install/), run `save-channeler.exe doctor`. Use `play-ds3.bat` after setup. Cross-built zips are CLI-only; build on Windows (without `-tags nogui`) for the GUI.

### Build manually

```bash
git clone https://github.com/migzeetigglez/ashen-sync.git
cd ashen-sync
go build -o save-channeler ./cmd/save-channeler   # GUI needs CGO + OpenGL on that OS
```

### One-time setup

```bash
rclone config                 # create a remote; prefer a dedicated folder
./save-channeler init         # pick remote name + path (default: save-channeler)
./save-channeler doctor
```

## Usage

```bash
./save-channeler games
./save-channeler status ds3
./save-channeler pull ds3
./save-channeler push ds3
./save-channeler play ds3
./save-channeler watch
./save-channeler gui
```

Conflict resolution:

```bash
./save-channeler pull ds3 --prefer remote
./save-channeler push ds3 --prefer local
```

Games: `ds2` (Scholar of the First Sin), `ds3`, `ds2-vanilla`.

### Steam Deck

1. Desktop Mode: install rclone (static binary in `$HOME/bin` is fine).
2. `rclone config` + `save-channeler init`.
3. For Game Mode, add a non-Steam shortcut or game launch option that runs:
   `/path/to/save-channeler play ds3`
4. Optional: run `save-channeler watch` from Desktop Mode / autostart when you want idle sync without the launch wrapper.

### Config

- Linux/macOS: `~/.config/save-channeler/config.toml`
- Windows: `%APPDATA%\save-channeler\config.toml`
- Snapshots: `~/.local/share/save-channeler/snapshots/` (Windows: `%LOCALAPPDATA%\save-channeler\snapshots\`)

Legacy `~/.config/ashen-sync/` configs are still read if the new path is missing.

```toml
rclone_path = "rclone"
remote = "gdrive"
remote_path = "save-channeler"
snapshot_keep = 10
quiet_period_seconds = 15
pull_interval_seconds = 300
ui_scale = 1.5                 # GUI scale (1.0–3.0); also changeable in the app
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

Cloud clients often lock files while uploading. Dark Souls writes saves frequently; a lock mid-write can corrupt the slot. Save Channeler copies through a staging area via rclone and never mounts the live save directory as a cloud folder.

## License

MIT — see [LICENSE](LICENSE).

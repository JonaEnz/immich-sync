# Immich Sync

*In development.*

A service for uploading to your Immich server.
Linux only, could work on Windows with minor adjustments.

## Currently Working

- [x] Upload images to Immich
- [x] Scan local directories for new / updated images
- [x] Add images to albums by directories
- [x] Scan in the background in regular intervals
- [x] Download albums from Immich
- [ ] Delete images from Immich

## Installation

1. Install the systemd service from the `immich-sync.service` file.
Compile the binary and place it at the specified path.

2. Create the configuration file at `/etc/immich-sync/config.yaml`:

```yaml
watch: [] # 
schedule: 15 # Sync intervals in minutes
server: "" # Server url with trailing /api
apikey: "" # API key (<immich>/user-settings?isOpen=api-keys) 
deviceid: "" # Device name
```

## Usage

The service needs to be running for all commands excluding daemon and scan.
The user config is only used for those commands.

```md
A client for uploading images to Immich

Usage:
  immich-sync [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  daemon      Daemon mode, opens a unix socket for communication
  help        Help about any command
  scan        Scans for new images, uses the daemon if it is running
  status      Checks the status of the service daemon
  upload      Uploads image(s) to Immich
  watch       Adds or remove directories from scan

Flags:
      --config string   config file (default is $HOME/.config/immich-sync/config.yaml)
  -h, --help            help for immich-sync

Use "immich-sync [command] --help" for more information about a command.
```

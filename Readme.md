# Immich Sync

*In development.*

A service for uploading to your Immich server.
Linux only, could work on Windows with minor adjustments.

## Currently Working

- [x] Upload images to Immich
- [x] Scan local directories for new / updated images
- [x] Scan in the background in regular intervals
- [ ] Delete images from Immich
- [ ] Download images from Immich

## Installation

Install the systemd service from the `immich-sync.service` file
Create the configuration file at `/etc/immich-sync/config.yaml`:

```yaml
watch: [] # 
schedule: 15 # Sync intervals in minutes
server: "" # Server url with trailing /api
apikey: "" # API key (<immich>/user-settings?isOpen=api-keys) 
deviceid: "" # Device name
```
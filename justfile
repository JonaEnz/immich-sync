update:
  wget https://raw.githubusercontent.com/immich-app/immich/refs/heads/main/open-api/immich-openapi-specs.json -O oapi/api.json
  go generate ./...
build:
  go build -o ./immich-sync
build-and-run: build
  sudo systemctl stop immich-sync
  sudo cp immich-sync /usr/bin
  sudo systemctl start immich-sync

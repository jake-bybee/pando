# Pando

Pando is a lightweight Go service that continuously backs up Immich assets
across a set of Pando peers. Every peer runs the same binary. Peers use
deterministic hashing to independently decide which peers store each asset,
so no shared database or central placement service is required.

The core backup flow is usable:

1. A peer starts and validates its Immich connection and local storage.
2. Peers register with the configured master peer and exchange the peer list.
3. Each peer polls Immich for assets.
4. Consistent hashing assigns each asset to `NUM_REPLICAS` peers.
5. Assigned assets are downloaded locally and recorded in a manifest.
6. The process repeats every minute after the current asset scan completes.

## Requirements

- Go 1.27.1 or newer
- A reachable Immich server
- An Immich API key with sufficient access (an `all` permission works)
- One or more directories with enough free space for the assigned assets
- Network connectivity between every Pando peer

## Configuration

Pando reads configuration from environment variables and an optional `.env`
file in the working directory.

| Variable | Required | Description |
| --- | --- | --- |
| `IMMICH_URL` | Yes | Base URL of the Immich server, for example `http://immich:2283` |
| `IMMICH_API_TOKEN` | Yes | Immich API key |
| `SELF_URL` | Yes | URL other peers use to reach this peer, for example `http://pando-1:8080` |
| `BACKUP_FOLDER_PATH` | Yes | Existing directory where this peer stores downloaded assets |
| `MASTER_PANDO_URL` | Recommended | URL of the peer used for initial registration |
| `MASTER_PANDO_PORT` | Conditional | Port used to derive `MASTER_PANDO_URL` when it is not set |
| `NUM_PEERS` | Yes | Expected total number of peers in the cluster |
| `NUM_REPLICAS` | Yes | Number of peers that should store each asset |
| `LISTEN_PORT` | No | HTTP port for this peer; defaults to `8080` |
| `TIME_ZONE` | No | IANA time zone for timestamps; defaults to `UTC` |

`NUM_REPLICAS` should be no greater than `NUM_PEERS`. The service waits until
the expected number of peers is registered and healthy before downloading
assets.

Example `.env` for a three-peer cluster:

```dotenv
IMMICH_URL=http://immich:2283
IMMICH_API_TOKEN=replace-with-an-immich-api-key

SELF_URL=http://pando-1:8080
MASTER_PANDO_URL=http://pando-1:8080
LISTEN_PORT=8080

BACKUP_FOLDER_PATH=/var/lib/pando
NUM_PEERS=3
NUM_REPLICAS=2
TIME_ZONE=UTC
```

Use a unique `SELF_URL`, `LISTEN_PORT`, and `BACKUP_FOLDER_PATH` on every peer.
The backup directory must exist before startup.

## Running

Build and run the service:

```sh
go build -o pando .
./pando
```

Or run it directly during development:

```sh
go run .
```

On startup, Pando performs an Immich API-key check, verifies the backup
directory, checks available storage against the Immich user's quota, creates
or loads a persistent peer ID, and starts the HTTP server. The peer ID is
stored in `settings/id.txt` below `BACKUP_FOLDER_PATH`.

The service also reconciles the downloaded-asset manifest at startup. Files
already represented in the manifest are retained, missing files are removed
from the manifest, and asset-ID-named files not yet represented are looked up
in Immich so their original filenames can be registered.

The service logs progress to standard output. It does not daemonize or manage
process supervision; use the service manager appropriate for your host when
running it continuously.

## HTTP API

The peer HTTP server listens on `LISTEN_PORT`.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Re-runs the Immich and local storage health checks |
| `GET` | `/manifest` | Lists assets recorded as downloaded by this peer |
| `POST` | `/registerPeer` | Registers a peer with this node |
| `POST` | `/registerPeersReceived` | Registers a relayed peer list |
| `GET` | `/relayPeers` | Returns the peers known by this node |

Peer registration endpoints are intended for communication between trusted
Pando nodes and are not authenticated by the service.

### Manifest pagination

`GET /manifest` accepts:

- `limit`: optional integer from `1` to `1000`; defaults to `100`
- `after`: optional asset ID cursor

Entries are sorted by asset ID. Use `nextCursor` as the next `after` value
until it is `null`.

```json
{
  "items": [
    { "assetId": "asset-id-1", "fileName": "photo-one.jpg" }
  ],
  "total": 1250,
  "nextCursor": "asset-id-1"
}
```

The manifest is stored at
`<BACKUP_FOLDER_PATH>/settings/asset-manifest.json`. It records successful
downloads; it is not a filesystem integrity or re-scan report. Downloaded
assets are stored in the backup root using their Immich asset ID as the
filename, with the original file extension preserved. Each archive is
extracted into a hash-named temporary directory and removed after its files
are moved into the backup root.
The manifest retains each asset's original Immich filename separately from
the asset-ID-based filename on disk.

## How replication works

For each Immich asset, every peer calculates the same score for every known
peer using the asset ID and peer ID. The highest-scoring peers are selected
until `NUM_REPLICAS` copies are assigned. Since the assignment is
deterministic, peers can download their own assignments without a coordinator.

Asset discovery is cursor-based and paginated through Immich. Failed downloads
reset the cursor so the scan can be retried. Once a scan completes, the service
waits one minute before polling again.

## Development

Run the unit tests:

```sh
go test ./...
```

Build the binary:

```sh
go build ./...
```

The repository also includes a `mise` task for starting multiple local
processes:

```sh
mise run test
```

## Current scope

Implemented:

- Immich connectivity and API-key health checks
- Local storage/quota health checks
- Persistent peer IDs
- Peer registration, relay, and health status
- Deterministic multi-peer asset placement
- Cursor-based Immich asset polling
- Bulk asset downloads
- Atomic downloaded-asset manifest updates
- Paginated manifest API

The service currently assumes a trusted network between peers. Peer API
authentication, encrypted peer-to-peer transport, filesystem re-verification,
and active rebalancing after membership changes are not implemented.

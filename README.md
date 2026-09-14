# Pando

Distributed immich photos backup binary

> 🚧 **Status: Work in progress.** Core replication is being actively built. Not yet production-ready — use at your own risk.

Named after [Pando](https://en.wikipedia.org/wiki/Pando_(tree)), the quaking aspen colony in Utah that presents as thousands of individual trees but is actually one connected root system. Losing individual stems doesn't kill the organism — that's the exact fault-tolerance property this project is built around.

## What it does

Pando runs the same lightweight Go agent on every node in your network. One node (wherever your source app — e.g. Immich — lives) watches for new or changed files. Using consistent hashing, every node can independently compute which files it's responsible for holding, with no central coordinator, no shared database, and no single point of failure for placement decisions.

Each file is replicated to N of your total nodes (configurable). No file is split or sharded — every replica is a complete, whole copy.

## Why

Self-hosted apps like Immich and Nextcloud store files locally by default. If that box dies, your data is gone unless you're manually backing it up. Pando solves this by spreading replicated copies across your other machines — including off-site nodes — automatically and continuously.

## Benefits

- **No single point of failure** — any node (except the one detecting new files, for now) can go down without losing data or breaking the system
- **Survives multi-node failure** — as long as at least one replica of a file survives, the file survives
- **Off-site protection** — geographically distant nodes protect against local disasters (power loss, fire, hardware failure) that could take out multiple local machines at once
- **No manual placement logic** — consistent hashing means every node can independently determine what it should hold, with zero coordination overhead
- **Self-healing (planned)** — periodic reconciliation will catch anything missed by real-time watching, including recovery after extended node downtime

## Roadmap

- [x] Design: consistent hashing for file placement
- [ ] Phase 1: Real-time file watching + replication (in progress)
- [ ] Phase 2: Periodic reconciliation pass
- [ ] Phase 3: Node failure detection (gossip-based health checks)
- [ ] Phase 4: Quorum-based writes + conflict resolution



*Built as a personal infrastructure project and a deep-dive into distributed systems fundamentals (replication, consistent hashing, failure detection, quorum consensus).*

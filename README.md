# VaultKV

VaultKV is a high-performance, distributed Key-Value store built entirely from scratch in Go. This project serves as an exploration of the deep internal mechanics behind modern database storage engines, specifically focusing on building a durable Log-Structured Merge (LSM) Tree architecture.

## Architecture & Features

VaultKV is designed around the core principles of high-throughput storage systems like RocksDB and Apache Cassandra.

### 1. Robust Durability (WAL)

To guarantee that no data is ever lost, VaultKV implements a custom **Write-Ahead Log (WAL)**.

- **Custom Binary Format**: Log entries are serialized into a highly optimized binary format.
- **Atomic CRC32 Checksums**: Every payload is verified against an IEEE CRC32 checksum to mathematically guarantee data integrity and catch torn writes from sudden hardware failures.
- **Record Framing (Upcoming)**: Every entry is prefixed with its total length. If the server loses power mid-write, the database can safely isolate the corrupted byte stream on reboot and recover the rest of the file without crashing.

### 2. LSM Tree Storage Engine (In Progress)

The core storage engine is actively being refactored into a full LSM Tree to optimize for massive write throughput.

- **MemTable (Next Step)**: Replacing the standard Go map with an in-memory, concurrent **Skip List** to keep incoming keys strictly ordered.
- **SSTables (Upcoming)**: Flushing the ordered MemTable to disk into immutable Sorted String Tables for efficient binary-search reads.
- **Compaction (Upcoming)**: Background goroutines to periodically merge SSTables and remove deleted keys (tombstones).

### 3. Distributed Cluster

VaultKV operates as a multi-node cluster to guarantee high availability.

- **Nodes**: Independent storage instances, each maintaining its own dedicated WAL and memory state.
- **Router**: A lightweight HTTP proxy that load-balances client requests across the active nodes in the cluster.

---

## API Usage

The cluster exposes a simple, universal HTTP JSON API.

**SET a Key-Value Pair**

```bash
curl -X POST http://localhost:8080/set \
     -H "Content-Type: application/json" \
     -d '{"key": "user:1", "value": "Alice"}'
```

**GET a Value**

```bash
curl "http://localhost:8080/get?key=user:1"
```

## Running the Project

VaultKV includes PowerShell scripts to easily spin up a multi-node environment.

**Start the Cluster (1 Router, 3 Nodes):**

```powershell
.\start_cluster.ps1
```

**Run the Chaos Test:**
VaultKV includes a chaos engineering test suite. It bombards the cluster with thousands of requests per second while randomly killing and restarting nodes. This physically verifies that the WAL successfully catches torn writes and recovers the database state without data loss.

```powershell
.\test\chaos_test.ps1
```

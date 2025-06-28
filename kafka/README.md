# Kafka-Style Log

This document outlines the implementation of a single-node, Kafka-style log service designed to meet the requirements of the [Fly.io Distributed Systems Challenge #5a](https://fly.io/dist-sys/5a/). The service provides an append-only log that handles `send`, `poll`, `commit_offsets`, and `list_committed_offsets` RPCs.

## Implementation

The core of the system is the `KafkaSever` struct, which manages the logs, offsets, and ensures thread-safe access to the data.

```go
type KafkaSever struct {
	log       map[string][]Message
	logOffset map[string]int
	lock      *sync.RWMutex
}
```

-   `log map[string][]Message`: This map stores the log messages. The key is the log topic (e.g., `"k1"`), and the value is a slice of `Message` structs. This append-only structure ensures that messages are stored in the order they are received.
-   `logOffset map[string]int`: This map tracks the last committed offset for each log topic. It is updated by the `commit_offsets` RPC and read by the `list_committed_offsets` RPC.
-   `lock *sync.RWMutex`: A read/write mutex is used to manage concurrency. It allows multiple concurrent reads (`poll`, `list_committed_offsets`) but ensures that write operations (`send`, `commit_offsets`) have exclusive access, preventing race conditions and data corruption.

## RPC Handlers

The implementation provides handlers for the four required RPCs, ensuring that the service behaves as expected.

### `Send`

The `Send` method handles requests to append a message to a log. It acquires a write lock, determines the new offset by taking the current length of the log slice, and appends the new message. This guarantees that offsets are unique and monotonically increasing for each log.

### `Poll`

The `Poll` method allows clients to retrieve messages from one or more logs starting from a given offset. It acquires a read lock and, for each requested log, returns a slice of messages from the specified offset to the end of the log.

### `CommitOffsets`

The `CommitOffsets` method is used by clients to signal that they have processed messages up to a certain offset. It acquires a write lock and updates the `logOffset` map with the new committed offsets provided by the client.

### `ListCommitedOffsets`

The `ListCommitedOffsets` method allows clients to query the last committed offset for a set of logs. It acquires a read lock and returns the stored offsets from the `logOffset` map. This enables consumers to know where to resume processing from.
# Kafka-Style Log

This document outlines the implementation of a distributed Kafka-style log service designed to meet the requirements of the [Fly.io Distributed Systems Challenge 5](https://fly.io/dist-sys/5a/). The service provides an append-only log that handles `send`, `poll`, `commit_offsets`, and `list_committed_offsets` RPCs using Maelstrom's key-value stores for persistence and consistency.

## Implementation

The core of the system is the `KafkaSever` struct, which uses Maelstrom's distributed key-value stores for data persistence and consistency guarantees.

```go
type KafkaSever struct {
	log       map[string][]Message
	logOffset map[string]int
	lock      *sync.RWMutex
	linKV     *maelstrom.KV
	seqKV     *maelstrom.KV
}
```

-   `linKV *maelstrom.KV`: A linearizable key-value store used for storing log messages. This provides strong consistency guarantees across the distributed system.
-   `seqKV *maelstrom.KV`: A sequentially consistent key-value store used for tracking committed offsets. This allows for efficient offset management while maintaining consistency.
- The in-memory fields `log` and `lock` are used for storing local states of keys that we are responsible for updating. `logOffset` is just for making the previously used [tests](./lib_test.go) for single node pass the build

## RPC Handlers

The implementation provides handlers for the four required RPCs, ensuring that the service behaves as expected.

### `Send`

The `Send` method handles requests to append a message to a log using atomic compare-and-swap operations:

1. Reads the current log messages from the linearizable KV store
2. Uses `CompareAndSwap` to atomically append the new message with the correct offset
3. Retries on precondition failures to handle concurrent writes
4. Returns the assigned offset on success

This approach ensures atomicity and prevents race conditions in a distributed environment.

>[!NOTE]
>
>We shard the keys so if node isn't the parent of that particular key we forward the message to its parent node and await its response which we then reply back to the client. This ensures less contention on the keys as every key is handled by one node so no chance of failure.

### `Poll`

The `Poll` method retrieves messages from distributed storage:

1. For each requested log key, reads the message array from the linearizable KV store
2. Returns messages starting from the specified offset
3. Handles missing keys gracefully by skipping them
4. Provides consistent reads across the distributed system

### `CommitOffsets`

The `CommitOffsets` method updates committed offsets using sequential consistency:

1. For each offset update, reads the current committed offset from the sequential KV store
2. Only updates if the new offset is greater than the existing one
3. Uses `CompareAndSwap` to atomically update offsets
4. Retries on precondition failures to handle concurrent updates

This prevents offsets from moving backwards while ensuring consistency.

### `ListCommitedOffsets`

The `ListCommitedOffsets` method queries committed offsets from distributed storage:

1. Reads committed offsets from the sequential KV store for each requested key
2. Handles missing keys by skipping them in the response
3. Returns a map of key-offset pairs for existing committed offsets

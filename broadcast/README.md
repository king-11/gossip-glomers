# Broadcast Server

The Broadcast Server is a distributed system component that implements a gossip protocol for message dissemination. It uses a concurrent map (`sync.Map`) for atomic operations and provides mechanisms to manage goroutines and channels effectively.

## Key Features

1. **Gossip Protocol**:
   - The server periodically gossips messages to a configurable number of random nodes.
   - Gossip frequency and the number of nodes to gossip to can be toggled using constants `GOSSIP_FREQUENCY` and `GOSSIP_NODES_COUNT` in the implementation.

2. **Concurrent Map**:
   - The server uses `sync.Map` for storing messages. This ensures atomic operations like compare-and-swap, avoiding race conditions.
   - This approach eliminates the need for explicit locking mechanisms like mutexes, simplifying the code and improving performance.

3. **Context for Goroutine Management**:
   - Context is used to manage the lifecycle of goroutines and channels.
   - When the server is stopped, the context is canceled, ensuring that all goroutines exit gracefully.
   - This prevents goroutines from being blocked indefinitely on I/O operations, especially when a `SIGTERM` signal is received.

4. **Channel Management**:
   - The server uses a buffered channel (`passingChannel`) to queue messages for broadcasting to neighbors.
   - Before accessing the channel, the implementation checks whether the channel is closed to avoid blocking goroutines.

## Implementation Details

### Gossiping
- The `Gossiper` function periodically selects a random subset of nodes and sends them the current list of messages using a new `GossipMessage` type. This message encapsulates all the messages stored in the server for efficient dissemination.
- The frequency of gossiping is controlled by the `GOSSIP_FREQUENCY` constant.
- The number of nodes to gossip to is controlled by the `GOSSIP_NODES_COUNT` constant.

### Message Storage
- Messages are stored in a `sync.Map`, which provides thread-safe operations.
- This eliminates the need for explicit locking mechanisms like mutexes, simplifying the code and improving performance.

### Context Usage
- A `context.Context` is passed to goroutines like `SendToNeighbours` and `Gossiper`.
- When the context is canceled, these goroutines terminate gracefully, ensuring no resources are leaked.

### Channel Management and Efficient Goroutine Usage
- The `passingChannel` is a buffered channel used to queue messages for broadcasting to neighbors. This ensures that messages are processed asynchronously without blocking the main execution flow.
- The `SendToNeighbours` function consumes messages from the `passingChannel` and efficiently uses goroutines to broadcast each message to all neighbors except the source node. A `sync.WaitGroup` is used to wait for all goroutines to complete before processing the next message.
- The `passingChannel` is closed when the server stops. Before sending or receiving from the channel, the implementation checks whether the channel is closed to avoid blocking operations.

## Learnings
Initially, a normal map was used for storing messages. However, concurrent access led to race conditions, especially during rehashing and table growth. A mutex was introduced to synchronize access to the map, but this added complexity and potential bottlenecks. The implementation was later improved by switching to `sync.Map`, which provides built-in support for concurrent access and atomic operations.

We were also facing `[runnable]` errors as we weren't stopping `SendToNeighbours` on `SIGTERM` as it was blocked on a closed channel using `select`. This was fixed by switching back to a `for` loop which stops on channel close automatically when `Stop` is called.

For the gossip ticker management, calling a stop on ticker doesn't close the channel, we initially added an additional `gossipTickerDone` channel. This was fine but we can utilize a more elegant way of `context` passing.

## Configuration
- **Gossip Frequency**: Modify the `GOSSIP_FREQUENCY` constant in `main.go` to change how often gossiping occurs.
- **Number of Nodes to Gossip To**: Modify the `GOSSIP_NODES_COUNT` constant in `main.go` to change the number of nodes selected for gossiping.

## Stopping the Server
- The `Stop` method ensures that all resources are cleaned up:
  - Closes the `passingChannel`.
  - Stops the gossip ticker.
  - Cancels the context to terminate all goroutines.

This implementation ensures a robust and efficient broadcast server that handles concurrency and resource management effectively.

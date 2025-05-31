# Broadcast Server

The Broadcast Server is a distributed system component that implements a gossip protocol for message dissemination. It uses a concurrent map (`sync.Map`) for atomic operations and provides mechanisms to manage goroutines and channels effectively. The server employs an optimized heap-like topology structure to minimize message propagation latency.

## Key Features

1. **Gossip Protocol**:
   - The server periodically gossips messages to a configurable number of random nodes.
   - Gossip frequency and the number of nodes to gossip to can be toggled using constants `GOSSIP_FREQUENCY` and `GOSSIP_NODES_COUNT` in the implementation.

2. **Optimized Topology Structure**:
   - The server implements a heap-like neighbor structure to minimize the maximum latency for message propagation.
   - Each node with ID "nX" connects to child nodes "n(2X % N)" and "n((2X + 1) % N)", where N is the total number of nodes.
   - This approach ensures message propagation with optimal distribution, using modulo arithmetic to wrap connections within the node range.
   - The wrapping approach guarantees that every node has exactly two neighbors, improving resiliency and ensuring consistent network density.

3. **Concurrent Map**:
   - The server uses `sync.Map` for storing messages. This ensures atomic operations like compare-and-swap, avoiding race conditions.
   - This approach eliminates the need for explicit locking mechanisms like mutexes, simplifying the code and improving performance.

4. **Context for Goroutine Management**:
   - Context is used to manage the lifecycle of goroutines and channels.
   - When the server is stopped, the context is canceled, ensuring that all goroutines exit gracefully.
   - This prevents goroutines from being blocked indefinitely on I/O operations, especially when a `SIGTERM` signal is received.

5. **Message Batching**:
   - The server uses a buffered channel (`passingChannel`) to queue messages for broadcasting to neighbors.
   - Messages are accumulated in a batch and sent to all neighbors at each tick, rather than sending immediately on arrival.
   - This batching is handled in the `SendToNeighbours` function, which locks the batch, sends it, and then clears it for the next interval.
   - Before accessing the channel, the implementation checks whether the channel is closed to avoid blocking goroutines.

## Implementation Details

### Gossiping
- The `Gossiper` function periodically selects a random subset of nodes and sends them the current list of messages using a new `GossipMessage` type. This message encapsulates all the messages stored in the server for efficient dissemination.
- The frequency of gossiping is controlled by the `GOSSIP_FREQUENCY` constant.
- The number of nodes to gossip to is controlled by the `GOSSIP_NODES_COUNT` constant.

### Message Storage
- Messages are stored in a `sync.Map`, which provides thread-safe operations.
- This eliminates the need for explicit locking mechanisms like mutexes, simplifying the code and improving performance.

### Topology Management
- When a topology message is received, the server calculates its neighbors using a heap-like structure.
- For a node with ID "nX", its neighbors are set to "n(2X % N)" and "n((2X + 1) % N)" where X is the numeric part and N is the total node count.
- This implementation uses modulo arithmetic to wrap node connections within the valid node range.
- The modulo approach ensures all nodes have exactly two neighbors, even when the calculated child IDs would exceed the number of nodes in the system.
- A dedicated `getNeighbours()` helper method encapsulates this logic, making it easy to modify or extend in the future.

### Context Usage
- A `context.Context` is passed to goroutines like `SendToNeighbours` and `Gossiper`.
- When the context is canceled, these goroutines terminate gracefully, ensuring no resources are leaked.

### Channel Management and Efficient Goroutine Usage
- The `passingChannel` is a buffered channel used to queue messages for broadcasting to neighbors. This ensures that messages are processed asynchronously without blocking the main execution flow.
- The `SendToNeighbours` function consumes messages from the `passingChannel`, batches them, and efficiently uses goroutines to broadcast each batch to all neighbors except the source node. A `sync.WaitGroup` is used to wait for all goroutines to complete before processing the next batch.
- The `passingChannel` is closed when the server stops. Before sending or receiving from the channel, the implementation checks whether the channel is closed to avoid blocking operations.

## Learnings
Initially, a normal map was used for storing messages. However, concurrent access led to race conditions, especially during rehashing and table growth. A mutex was introduced to synchronize access to the map, but this added complexity and potential bottlenecks. The implementation was later improved by switching to `sync.Map`, which provides built-in support for concurrent access and atomic operations.

We were also facing `[runnable]` errors as we weren't stopping `SendToNeighbours` on `SIGTERM` as it was blocked on a closed channel using `select`. This was fixed by switching back to a `for` loop which stops on channel close automatically when `Stop` is called.

For the gossip ticker management, calling a stop on ticker doesn't close the channel, we initially added an additional `gossipTickerDone` channel. This was fine but we can utilize a more elegant way of `context` passing.

In topology, we initially tried using circular with front, back and an opposite in the circle node as neighbours but that didn't work out well and had increased tail latency in some cases. Switching to tree like structure gave things more than I expected probably due to *logarithmic* complexity of a tree in passing around the message whereas circular was *linear*.

We started batching messages together when sending to neibhours instead of sending a broadcast message for each new message that a node receives. This reduces the number of total messages sent over the network but a huge margin. We can adjust the frequency when we send messages to neighbours an alternate approach can be to check the size of batch and then send.

## Configuration
- **Neighbor Gossip Frequency:** Modify the `neighboursTickDuration` parameter to change how often batches are sent to neighbors.
- **Random-Node Gossip Frequency:** Modify the `gossipTickDuration` parameter to change how often the full message set is gossiped to random nodes.
- **Number of Nodes to Gossip To:** Adjust the random node count parameter in `main.go` to change the number of nodes selected for random gossiping.

## Stopping the Server
- The `Stop` method ensures that all resources are cleaned up:
  - Closes the `passingChannel`.
  - Stops the gossip ticker.
  - Cancels the context to terminate all goroutines.

This implementation ensures a robust and efficient broadcast server that handles concurrency and resource management effectively, with optimized batching for neighbor gossip and periodic global dissemination via random-node gossip.

# Broadcast Server

The Broadcast Server is a distributed system component that implements a gossip protocol for message dissemination. It uses a concurrent map (`sync.Map`) for atomic operations and provides mechanisms to manage goroutines and channels effectively. The server employs an optimized heap-like topology structure to minimize message propagation latency.

I have written a dedicated walkthrough of my solution on my [blog](https://blog.king-11.dev/posts/efficient-gossip-distributed-systems/).

## Learnings
Initially, a normal map was used for storing messages. However, concurrent access led to race conditions, especially during rehashing and table growth. A mutex was introduced to synchronize access to the map, but this added complexity and potential bottlenecks. The implementation was later improved by switching to `sync.Map`, which provides built-in support for concurrent access and atomic operations.

I was also facing `[runnable]` errors as I wasn't stopping `SendToNeighbours` on `SIGTERM` as it was blocked on a closed channel using `select`. This was fixed by switching back to a `for` loop which stops on channel close automatically when `Stop` is called.

For the gossip ticker management, calling a stop on ticker doesn't close the channel, I initially added an additional `gossipTickerDone` channel. This was fine but I can utilize a more elegant way of `context` passing.

In topology, I initially tried using circular with front, back and an opposite in the circular topology as neighbours but that didn't work out well and had increased tail latency in some cases. Switching to tree like structure gave things more than I expected probably due to *logarithmic* complexity of a tree in passing around the message whereas circular was *linear*.

I started batching messages together when sending to neibhours instead of sending a broadcast message for each new message that a node receives. This reduces the number of total messages sent over the network but a huge margin. I can adjust the frequency when a node sends messages to neighbours, an alternate approach can be to check the size of batch and then send.

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

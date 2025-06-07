# Grow-Only Counter

The Grow-Only Counter is a distributed counter that maintains a monotonically increasing value across multiple nodes. It uses Maelstrom's sequential key-value store with compare-and-swap operations to ensure consistent updates in a distributed environment.

## Key Features

1. **Sequential Key-Value Store**:
  - Uses Maelstrom's `SeqKV` for sequential consistency guarantees.

2. **Compare-and-Swap (CAS) Operations**:
  - The `Add` operation uses CAS to atomically increment the counter value.
  - Implements retry logic with optimistic concurrency control to handle concurrent updates.
  - Uses a loop with context cancellation to gracefully handle timeouts and cancellations.

3. **Read Operations**:
  - Uses `Write` operation before every read to ensure freshness of reads.

## Implementation Details

### Counter Storage
- The counter value is stored under a single key (`GROW_ONLY_KEY = "groww"`) in the sequential KV store.
- The sequential KV store provides linearizable reads and writes, ensuring strong consistency.

### Add Operation
- Implements an optimistic concurrency control pattern:
  1. Read the current value from the KV store
  2. Calculate the new value by adding the delta
  3. Use `CompareAndSwap` to atomically update if the value hasn't changed
  4. If CAS fails (value changed), retry the entire operation
- Handles the case where the key doesn't exist initially (treats as value 0)
- Uses context cancellation to break out of retry loops when needed

### Read Operation
- Write a unqiue key generated using `rand` package, before reading the counter value from the KV store.
- Simple read operation that fetches the current counter value from the KV store
- Returns the current value directly without any caching or local state

## Configuration
- **Counter Key:** Modify the `GROW_ONLY_KEY` constant to change the key used for storing the counter value

## Learning

The only failure I encountered was missing out that the key might not be present initially which can lead to just everything breaking up. So when write a new key in a greedy manner only when we receive a new `Add` operation. The `Read` operations before this all just return the default value of `0`.

So running the workload multiple time I encountered an error which made by understanding for Sequential KV better. The check that workload does is last read should have the latest value of counter everything should have converged at this point. But the reads from a `SeqKV` can return stale values.

To counteract that the [suggestion](https://github.com/jepsen-io/maelstrom/issues/39#issuecomment-1445414521) was to perform write of a unique key which creates a new state of `SeqKV` that can't exist until all the writes before it are completed. This ensures all our writes get the latest data. The other suggestion was to do multiple of these operations which probablistically ensures that we would get the latest value at some point of time.

There was also try to use [identity CAS](https://github.com/jepsen-io/maelstrom/issues/39#issuecomment-1445195514) for freshness but a CAS operation can also be stale that doesn't make things fresh as CAS can be reordered to some point in time.

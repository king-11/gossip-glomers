# Unique Id Generation

This implementation generates unique IDs using a combination of the current epoch time, source node ID, destination node ID, and a counter. The approach ensures that IDs are unique across distributed nodes.

## Approach

1. **Epoch Time**: The current Unix timestamp (in seconds) is used as the base for the unique ID. This ensures that IDs are time-ordered.
2. **Source Node ID**: The ID of the source node is extracted and encoded into the unique ID.
3. **Destination Node ID**: The ID of the destination node is also encoded into the unique ID.
4. **Counter**: A 16-bit counter is used to differentiate IDs generated within the same second.

The unique ID is constructed as a 64-bit integer with the following structure:
- **Bits 63-32**: Epoch time (32 bits)
- **Bits 31-24**: Source node ID (8 bits)
- **Bits 23-16**: Destination node ID (8 bits)
- **Bits 15-0**: Counter (16 bits)

## Example

For example, if:
- Epoch time = `1697040000` (Unix timestamp)
- Source node ID = `c1` (converted to integer `1`)
- Destination node ID = `n2` (converted to integer `2`)
- Counter = `42`

The unique ID would be calculated as:

```cpp
Unique ID = (1697040000 << 32) | (1 << 24) | (2 << 16) | 42
          = 72866491809705994
```

This ensures that the ID is unique and encodes the relevant information for traceability.

## Additional Notes

- The counter is a 16-bit value and increments with each ID generation. If the counter exceeds 65,535 IDs within a single second (i.e., at a rate higher than 65,535 QPS), it will overflow and wrap back to 0. This could result in duplicate IDs if the epoch time, source node ID, and destination node ID remain the same.
- The source and destination node IDs are extracted by trimming specific prefixes (`c` for source and `n` for destination) and converting the remaining part to integers.

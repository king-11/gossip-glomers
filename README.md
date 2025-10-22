# Distributed Systems Challenges

## About

The challenges are built on top of a platform called Maelstrom, which in turn, is built on Jepsen.
This platform lets you build out a "node" in your distributed system and Maelstrom will handle the routing of messages between the those nodes.
This lets Maelstrom inject failures and perform verification checks based on the consistency guarantees required by each challenge.

## Why

It's no secret that distributed systems are infuriating and difficult.
Even the best developers in the world can be brought to their knees in the face of cluster failures.

I have published the walkthrough of my solutions on my **[blog](https://blog.king-11.dev/tags/distributed-system/)**.

## Progress

|Sr. No. |Name |Status|
|:---:|:---:|:---:|
|[1](https://fly.io/dist-sys/1) |Echo |🌟 |
|[2](https://fly.io/dist-sys/2/) |Unique ID Generation |🌟 |
|[3a](https://fly.io/dist-sys/3a/) |Single-Node Broadcast |🌟 |
|[3b](https://fly.io/dist-sys/3b/) |Multi-Node Broadcast |🌟 |
|[3c](https://fly.io/dist-sys/3c/) |Fault Tolerant Broadcast |🌟 |
|[3d](https://fly.io/dist-sys/3d/) |Efficient Broadcast, Part I |🌟 |
|[3e](https://fly.io/dist-sys/3e/) |Efficient Broadcast, Part II |🌟 |
|[4](https://fly.io/dist-sys/4/) |Grow-Only Counter |🌟 |
|[5a](https://fly.io/dist-sys/5a/) |Single-Node Kafka-Style Log |🌟 |
|[5b](https://fly.io/dist-sys/5b/) |Multi-Node Kafka-Style Log |🌟 |
|[5c](https://fly.io/dist-sys/5c/) |Efficient Kafka-Style Log |🌟 |
|[6a](https://fly.io/dist-sys/6a/) |Single-Node, Totally-Available Transactions |🌟 |
|[6b](https://fly.io/dist-sys/6b/) |Totally-Available, Read Uncommitted Transactions |🌟 |
|[6c](https://fly.io/dist-sys/6c/) |Totally-Available, Read Committed Transactions |🌟 |

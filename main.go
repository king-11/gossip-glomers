package main

import (
	"context"
	"encoding/json"
	"gossip-glomers/broadcast"
	"gossip-glomers/echo"
	uniqueidgeneration "gossip-glomers/unique-id-generation"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	GOSSIP_FREQUENCY   = 2 * time.Second
	GOSSIP_NODES_COUNT = 10
)

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		return echo.HandleEcho(msg, n)
	})

	s := uniqueidgeneration.NewUniqueIdServer(n)
	n.Handle("generate", func(msg maelstrom.Message) error {
		return s.HandleMessage(msg)
	})

	b := broadcast.NewBroadcastServer(n, GOSSIP_FREQUENCY)
	n.Handle("read", func(msg maelstrom.Message) error {
		body := new(broadcast.ReadMessage)
		if err := json.Unmarshal(msg.Body, body); err != nil {
			return err
		}

		return n.Reply(msg, b.Read(body))
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		body := new(broadcast.TopologyMessage)
		if err := json.Unmarshal(msg.Body, body); err != nil {
			return err
		}

		return n.Reply(msg, b.Topology(body))
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		body := new(broadcast.BroadcastMessage)
		if err := json.Unmarshal(msg.Body, body); err != nil {
			return err
		}

		reply, replyBack := b.Broadcast(body, msg.Src)
		if !replyBack {
			return nil
		}
		return n.Reply(msg, reply)
	})

	n.Handle("gossip", func(msg maelstrom.Message) error {
		body := new(broadcast.GossipMessage)
		if err := json.Unmarshal(msg.Body, body); err != nil {
			return err
		}

		b.Gossip(body.Messages, msg.Src)
		return nil
	})

	context, cancelContext := context.WithCancel(context.Background())
	go b.SendToNeighbours(context)
	go b.Gossiper(context, GOSSIP_NODES_COUNT)
	defer b.Stop()
	defer cancelContext()
	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

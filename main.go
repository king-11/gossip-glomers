package main

import (
	"context"
	"encoding/json"
	"gossip-glomers/broadcast"
	"gossip-glomers/echo"
	growonlycounter "gossip-glomers/grow-only-counter"
	uniqueidgeneration "gossip-glomers/unique-id-generation"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	GOSSIP_FREQUENCY     = 2 * time.Second
	NEIGHBOURS_FREQUENCY = 50 * time.Millisecond
	GOSSIP_NODES_COUNT   = 10
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

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()
	// b := BroadCastServerSetup(n, ctx)
	// defer b.Stop()

	gocs := growonlycounter.NewGrowOnlyCounterServer(n)
	n.Handle("read", func(msg maelstrom.Message) error {
		readMessage := new(growonlycounter.ReadMessage)
		if err := json.Unmarshal(msg.Body, readMessage); err != nil {
			return err
		}

		return n.Reply(msg, gocs.Read(readMessage, ctx))
	})

	n.Handle("add", func(msg maelstrom.Message) error {
		addMessage := new(growonlycounter.AddMessage)
		if err := json.Unmarshal(msg.Body, addMessage); err != nil {
			return err
		}

		return n.Reply(msg, gocs.Add(addMessage, ctx))
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

func BroadCastServerSetup(n *maelstrom.Node, ctx context.Context) broadcast.BroadcastServer {
	b := broadcast.NewBroadcastServer(n, GOSSIP_FREQUENCY, NEIGHBOURS_FREQUENCY)
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

	go b.SendToNeighbours(ctx)
	go b.Gossiper(ctx, GOSSIP_NODES_COUNT)

	return b
}

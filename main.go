package main

import (
	"encoding/json"
	"gossip-glomers/broadcast"
	"gossip-glomers/echo"
	uniqueidgeneration "gossip-glomers/unique-id-generation"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
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

	b := broadcast.NewBroadcastServer(n)
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

	go b.SendToNeighbours()
	go b.Gossiper()
	defer b.Stop()
	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

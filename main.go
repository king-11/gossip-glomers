package main

import (
	"context"
	"encoding/json"
	"gossip-glomers/broadcast"
	"gossip-glomers/echo"
	growonlycounter "gossip-glomers/grow-only-counter"
	kafka "gossip-glomers/kafka"
	totallyavailable "gossip-glomers/totally-available"
	uniqueidgeneration "gossip-glomers/unique-id-generation"
	"log"
	"os"
	"os/signal"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	GOSSIP_FREQUENCY     = 5 * time.Second
	NEIGHBOURS_FREQUENCY = 50 * time.Millisecond
	GOSSIP_NODES_COUNT   = 5
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

	ctx, cancelContext := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
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

	// k := KafkaNodeSetup(n, ctx)

	ta := totallyavailable.NewTotallyAvailableNode(n)
	go ta.WriteServer(ctx)
	n.Handle("txn", func(msg maelstrom.Message) error {
		txnMessage := new(totallyavailable.TxnRequest)
		if err := json.Unmarshal(msg.Body, txnMessage); err != nil {
			return err
		}

		return n.Reply(msg, ta.Transaction(txnMessage))
	})

	n.Handle("write", func(msg maelstrom.Message) error {
		writeMessage := new(totallyavailable.WriteMessage)
		if err := json.Unmarshal(msg.Body, writeMessage); err != nil {
			return err
		}

		ta.Write(writeMessage.Requests)
		return nil
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

func KafkaNodeSetup(n *maelstrom.Node, ctx context.Context) *kafka.KafkaSever {
	kafkaServer := kafka.NewKafkaSever(n)
	n.Handle("send", func(msg maelstrom.Message) error {
		sendMessage := new(kafka.SendMessage)
		if err := json.Unmarshal(msg.Body, sendMessage); err != nil {
			return err
		}

		return n.Reply(msg, kafkaServer.Send(sendMessage, ctx))
	})

	n.Handle("poll", func(msg maelstrom.Message) error {
		pollMessage := new(kafka.PollMessage)
		if err := json.Unmarshal(msg.Body, pollMessage); err != nil {
			return err
		}

		return n.Reply(msg, kafkaServer.Poll(pollMessage, ctx))
	})

	n.Handle("commit_offsets", func(msg maelstrom.Message) error {
		commitOffsets := new(kafka.CommitOffsets)
		if err := json.Unmarshal(msg.Body, commitOffsets); err != nil {
			return err
		}

		return n.Reply(msg, kafkaServer.CommitOffsets(commitOffsets, ctx))
	})

	n.Handle("list_committed_offsets", func(msg maelstrom.Message) error {
		listCommittedOffsets := new(kafka.ListCommittedOffsets)
		if err := json.Unmarshal(msg.Body, listCommittedOffsets); err != nil {
			return err
		}

		return n.Reply(msg, kafkaServer.ListCommitedOffsets(listCommittedOffsets, ctx))
	})

	return kafkaServer
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

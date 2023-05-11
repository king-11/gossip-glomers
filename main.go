package main

import (
	"context"
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node          *maelstrom.Node
	KV            *maelstrom.KV
}

func (s *Server) incrementCounter(delta int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	val, err := s.KV.ReadInt(ctx, "read")
	if err != nil {
		val = 0
	}

	err = s.KV.CompareAndSwap(ctx, "read", val, val + delta, true)
	if err != nil {
		for {
			val, _ := s.KV.ReadInt(ctx, "read")
			err = s.KV.CompareAndSwap(ctx, "read", val, val + delta, true)
			if err == nil {
				break
			}
		}
	}
}

func (s *Server) getCounter() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	val, err := s.KV.ReadInt(ctx, "read")
	if err != nil {
		val = 0
	}
	return val
}

func NewServer(node *maelstrom.Node, kv *maelstrom.KV) *Server {
	s := &Server{
		Node:          node,
		KV:            kv,
	}

	// Register the handlers
	node.Handle("add", s.AddHandler)
	node.Handle("read", s.ReadHandler)

	return s
}

func (s *Server) AddHandler(msg maelstrom.Message) error {
	type Body struct {
		MessageType string `json:"type"`
		Delta       int    `json:"delta"`
	}

	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.incrementCounter(body.Delta)

	return s.Node.Reply(msg, map[string]string{
		"type": "add_ok",
	})
}

func (s *Server) ReadHandler(msg maelstrom.Message) error {
	type Body struct {
		MessageType string `json:"type"`
	}

	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	return s.Node.Reply(msg, map[string]any{
		"type":  "read_ok",
		"value": s.getCounter(),
	})
}

func main() {
	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)
	s := NewServer(n, kv)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node     *maelstrom.Node
	UniqueID int
	Seen     map[int]any
	Topology map[string][]string
}

func NewServer(node *maelstrom.Node) *Server {
	s := &Server{
		Node:     node,
		UniqueID: 1,
		Seen:     make(map[int]any),
	}

	// Register the handlers
	node.Handle("echo", s.EchoHandler)
	node.Handle("generate", s.GenerateHandler)
	node.Handle("broadcast", s.BroadcastHandler)
	node.Handle("read", s.ReadHandler)
	node.Handle("topology", s.TopologyHandler)

	return s
}

func (s *Server) EchoHandler(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "echo_ok"

	return s.Node.Reply(msg, body)
}

func (s *Server) GenerateHandler(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "generate_ok"

	id := fmt.Sprintf("%d-%s-%s-%d", time.Now().Unix(), msg.Src, msg.Dest, s.UniqueID)
	s.UniqueID += 1

	body["id"] = id

	return s.Node.Reply(msg, body)
}

func (s *Server) BroadcastHandler(msg maelstrom.Message) error {
	type Body struct {
		TypeOf  string `json:"type"`
		Message int    `json:"message"`
	}

	var body Body
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.Seen[body.Message] = struct{}{}

	resp_body := make(map[string]any)
	resp_body["type"] = "broadcast_ok"

	return s.Node.Reply(msg, resp_body)
}

func (s *Server) ReadHandler(msg maelstrom.Message) error {
	body := make(map[string]any)
	vals := make([]int, 0, len(s.Seen))

	for val := range s.Seen {
		vals = append(vals, val)
	}

	body["type"] = "read_ok"
	body["messages"] = vals

	return s.Node.Reply(msg, body)
}

func (s *Server) TopologyHandler(msg maelstrom.Message) error {
	var body struct {
		Topology map[string][]string `json:"topology"`
	}

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.Topology = body.Topology

	body_send := make(map[string]string)
	body_send["type"] = "topology_ok"

	return s.Node.Reply(msg, body_send)
}

func main() {
	n := maelstrom.NewNode()
	s := NewServer(n)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}

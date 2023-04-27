package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node          *maelstrom.Node
	UniqueID      int
	Topology      map[string][]string
	TopologyMutex *sync.RWMutex
	seen          map[int]any
	seenMutex     *sync.RWMutex
}

func (s *Server) checkMessage(message int) bool {
	s.seenMutex.RLock()
	defer s.seenMutex.RUnlock()

	_, ok := s.seen[message]
	return ok
}

func (s *Server) addMessage(message int) {
	s.seenMutex.Lock()
	defer s.seenMutex.Unlock()

	s.seen[message] = struct{}{}
}

func (s *Server) seenNow() []int {
	s.seenMutex.RLock()
	defer s.seenMutex.RUnlock()

	vals := make([]int, 0, len(s.seen))

	for val := range s.seen {
		vals = append(vals, val)
	}

	return vals
}

func NewServer(node *maelstrom.Node) *Server {
	s := &Server{
		Node:          node,
		UniqueID:      1,
		Topology:      make(map[string][]string),
		TopologyMutex: &sync.RWMutex{},
		seen:          make(map[int]any),
		seenMutex:     &sync.RWMutex{},
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
		Message   int `json:"message"`
		MessageID int `json:"msg_id"`
	}

	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	if s.checkMessage(body.Message) {
		return nil
	}

	s.addMessage(body.Message)

	s.TopologyMutex.RLock()
	defer s.TopologyMutex.RUnlock()
	for _, node := range s.Topology[s.Node.ID()] {
		if node == msg.Src {
			continue
		}
		s.Node.Send(node, map[string]any{
			"type":    "broadcast",
			"message": body.Message,
		})
	}

	if body.MessageID != 0 {
		resp_body := make(map[string]any)
		resp_body["type"] = "broadcast_ok"

		return s.Node.Reply(msg, resp_body)
	}

	return nil
}

func (s *Server) ReadHandler(msg maelstrom.Message) error {
	body := make(map[string]any)

	vals := s.seenNow()

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

	s.TopologyMutex.Lock()
	defer s.TopologyMutex.Unlock()
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

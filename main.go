package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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

	return maps.Keys(s.seen)
}

func (s *Server) neighbours() []string {
	s.TopologyMutex.RLock()
	defer s.TopologyMutex.RUnlock()

	return s.Topology[s.Node.ID()]
}

func (s *Server) notNeighbours() []string {
	n := s.neighbours()
	slices.Sort(n)
	nn := make([]string, 0, len(s.Node.NodeIDs()))
	for _, node := range s.Node.NodeIDs() {
		if _, found := slices.BinarySearch(n, node); !found && node != s.Node.ID() {
			nn = append(nn, node)
		}
	}

	return nn
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

	rand.Seed(time.Now().Unix())

	// Register the handlers
	node.Handle("echo", s.EchoHandler)

	node.Handle("generate", s.GenerateHandler)

	node.Handle("broadcast", s.BroadcastHandler)
	node.Handle("read", s.ReadHandler)
	node.Handle("topology", s.TopologyHandler)
	node.Handle("gossip", s.GossipHandler)

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

	if !s.checkMessage(body.Message) {
		s.addMessage(body.Message)
		neighours := s.neighbours()
		wg := &sync.WaitGroup{}
		for _, node := range neighours {
			if node == msg.Src {
				continue
			}

			wg.Add(1)
			go func(dest string, wg *sync.WaitGroup) {
				s.Node.Send(dest, map[string]any{
					"type":    "broadcast",
					"message": body.Message,
				})
				wg.Done()
			}(node, wg)
		}
		wg.Wait()
	}

	nn := s.notNeighbours()
	randNode := nn[rand.Intn(len(nn))]
	go func(dest string, messages []int) {
		s.Node.Send(dest, map[string]any{
			"type":    "gossip",
			"message": messages,
		})
	}(randNode, s.seenNow())

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

func (s *Server) GossipHandler(msg maelstrom.Message) error {
	type Body struct {
		Message   []int `json:"message"`
	}

	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	notFound := make([]int, 0, len(body.Message))
	for _, val := range body.Message {
		if !s.checkMessage(val) {
			s.addMessage(val)
			notFound = append(notFound, val)
		}
	}

	if len(notFound) == 0 {
		return nil
	}

	wg := &sync.WaitGroup{}
	for _, node := range s.neighbours() {
		if node == msg.Src {
			continue
		}

		wg.Add(1)
		go func(dest string, wg *sync.WaitGroup) {
			s.Node.Send(dest, map[string]any{
				"type":    "gossip",
				"message": notFound,
			})
			wg.Done()
		}(node, wg)
	}
	wg.Wait()

	return nil
}

func main() {
	n := maelstrom.NewNode()
	s := NewServer(n)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}

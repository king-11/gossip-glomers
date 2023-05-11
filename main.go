package main

import (
	"context"
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
	KV            *maelstrom.KV
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
	// node.Handle("read", s.ReadBroadcastHandler)
	node.Handle("topology", s.TopologyHandler)
	node.Handle("gossip", s.GossipHandler)

	node.Handle("add", s.AddHandler)
	node.Handle("read", s.ReadCounterHandler)

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

	wg := &sync.WaitGroup{}
	if !s.checkMessage(body.Message) {
		s.addMessage(body.Message)
		neighours := s.neighbours()
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
	}

	curTime := time.Now().Second()
	if curTime%7 == 0 {
		nn := s.notNeighbours()
		randNode := nn[rand.Intn(len(nn))]

		wg.Add(1)
		go func(dest string, messages []int, wg *sync.WaitGroup) {
			s.Node.Send(dest, map[string]any{
				"type":    "gossip",
				"message": messages,
			})
			wg.Done()
		}(randNode, s.seenNow(), wg)
	}

	wg.Wait()

	if body.MessageID != 0 {
		resp_body := make(map[string]any)
		resp_body["type"] = "broadcast_ok"

		return s.Node.Reply(msg, resp_body)
	}

	return nil
}

func (s *Server) ReadBroadcastHandler(msg maelstrom.Message) error {
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
		Message []int `json:"message"`
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

func (s *Server) ReadCounterHandler(msg maelstrom.Message) error {
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

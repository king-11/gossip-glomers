package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Message struct {
	message float64
	offset  int
}

func NewMessage(m float64, o int) Message {
	return Message{message: m, offset: o}
}

type Server struct {
	Node                *maelstrom.Node
	LogMutex            *sync.RWMutex
	Log                 map[string][]Message
	GlobalOffestMutex   *sync.RWMutex
	GlobalOffset        map[string]int
	CommitedOffestMutex *sync.RWMutex
	CommitedOffset      map[string]int
}

func NewServer(node *maelstrom.Node) *Server {
	s := &Server{
		Node:                node,
		LogMutex:            &sync.RWMutex{},
		Log:                 make(map[string][]Message),
		GlobalOffestMutex:   &sync.RWMutex{},
		GlobalOffset:        make(map[string]int),
		CommitedOffestMutex: &sync.RWMutex{},
		CommitedOffset:      make(map[string]int),
	}

	// Register the handlers
	s.Node.Handle("init", s.InitHandler)
	s.Node.Handle("send", s.SendHandler)
	s.Node.Handle("poll", s.PollHandler)
	s.Node.Handle("commit_offsets", s.CommitOffset)
	s.Node.Handle("list_committed_offsets", s.ListCommitOffset)

	return s
}

func (s *Server) InitHandler(msg maelstrom.Message) error {
	f, err := os.OpenFile("/tmp/maelstrom.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	log.SetOutput(f)

	return nil
}

func (s *Server) SendHandler(msg maelstrom.Message) error {
	type Body struct {
		Key string  `json:"key"`
		Msg float64 `json:"msg"`
	}
	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.GlobalOffestMutex.Lock()
	defer s.GlobalOffestMutex.Unlock()
	offest, ok := s.GlobalOffset[body.Key]
	if !ok {
		offest = 0
	}
	s.GlobalOffset[body.Key] = offest + 1
	log.Printf("offest %d", offest)

	s.LogMutex.Lock()
	defer s.LogMutex.Unlock()
	logs, ok := s.Log[body.Key]
	if !ok {
		logs = make([]Message, 0, 1)
	}
	logs = append(logs, NewMessage(body.Msg, offest))
	s.Log[body.Key] = logs

	replyBody := map[string]any{
		"type":   "send_ok",
		"offset": offest,
	}
	return s.Node.Reply(msg, replyBody)
}

func (s *Server) PollHandler(msg maelstrom.Message) error {
	type Body struct {
		Offsets map[string]int `json:"offsets"`
	}
	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	log.Printf("offsets requested %v", body.Offsets)

	s.LogMutex.RLock()
	defer s.LogMutex.RUnlock()
	msgs := make(map[string][][2]any)
	for key, startIndex := range body.Offsets {
		for _, msg := range s.Log[key][startIndex:] {
			msgs[key] = append(msgs[key], [2]any{msg.offset, msg.message})
		}
		log.Printf("messages for key %s are: %v\n", key, msgs[key])
	}

	return s.Node.Reply(msg, map[string]any{
		"type": "poll_ok",
		"msgs": msgs,
	})
}

func (s *Server) CommitOffset(msg maelstrom.Message) error {
	type Body struct {
		Offsets map[string]int `json:"offsets"`
	}
	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.CommitedOffestMutex.Lock()
	defer s.CommitedOffestMutex.Unlock()
	for key, val := range body.Offsets {
		s.CommitedOffset[key] = val
	}

	return s.Node.Reply(msg, map[string]string{
		"type": "commit_offsets_ok",
	})
}

func (s *Server) ListCommitOffset(msg maelstrom.Message) error {
	type Body struct {
		Keys []string `json:"keys"`
	}
	body := new(Body)
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	offsets := make(map[string]int)
	s.CommitedOffestMutex.RLock()
	defer s.CommitedOffestMutex.RUnlock()
	for _, key := range body.Keys {
		if val, ok := s.CommitedOffset[key]; ok {
			offsets[key] = val
		}
	}

	return s.Node.Reply(msg, map[string]any{
		"type":    "list_committed_offsets_ok",
		"offsets": offsets,
	})
}

func main() {
	n := maelstrom.NewNode()
	s := NewServer(n)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}

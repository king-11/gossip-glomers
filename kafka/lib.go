package kafka

import (
	"log"
	"maps"
	"os"
	"sync"
)

type KafkaSever struct {
	log       map[string][]Message
	logOffset map[string]int
	lock      *sync.RWMutex
}

func NewKafkaSever() *KafkaSever {
	log.SetOutput(os.Stderr)
	return &KafkaSever{
		log:       make(map[string][]Message),
		logOffset: make(map[string]int),
		lock:      &sync.RWMutex{},
	}
}

func (s *KafkaSever) Send(msg *SendMessage) SendMessageReply {
	s.lock.Lock()
	defer s.lock.Unlock()

	var messages []Message
	var exists bool
	if messages, exists = s.log[msg.Key]; !exists {
		messages = make([]Message, 0, 1)
	}

	index := len(messages)
	messages = append(messages, NewMessage(index, msg.Value))
	s.log[msg.Key] = messages

	return msg.Reply(index)
}

func (s *KafkaSever) Poll(msg *PollMessage) PollMessageReply {
	s.lock.RLock()
	defer s.lock.RUnlock()

	messages := make(map[string][]Message)
	for key, offset := range msg.Offsets {
		if logMessages, exists := s.log[key]; !exists {
			log.Printf("key: %s not found in logs", key)
			continue
		} else if len(logMessages) < offset {
			log.Printf("size of key: %s was %d which is less than offset %d", key, len(logMessages), offset)
			continue
		}

		messages[key] = s.log[key][offset:]
	}

	return PollMessageReply{MessageType: "poll_ok", Messages: messages}
}

func (s *KafkaSever) CommitOffsets(msg *CommitOffsets) CommitOffsetsReply {
	s.lock.Lock()
	defer s.lock.Unlock()

	maps.Copy(s.logOffset, msg.Offsets)

	return msg.Reply()
}

func (s *KafkaSever) ListCommitedOffsets(msg *ListCommittedOffsets) ListCommittedOffsetsReply {
	s.lock.RLock()
	defer s.lock.RUnlock()

	offsets := make(map[string]int)
	for _, key := range msg.Keys {
		if offset, exists := s.logOffset[key]; !exists {
			log.Printf("log offset not found of key:%s", key)
			continue
		} else {
			offsets[key] = offset
		}
	}

	return ListCommittedOffsetsReply{MessageType: "list_committed_offsets_ok", Offsets: offsets}
}

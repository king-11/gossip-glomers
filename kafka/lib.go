package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type KafkaSever struct {
	log       map[string][]Message
	logOffset map[string]int
	lock      *sync.RWMutex
	linKV     *maelstrom.KV
	seqKV     *maelstrom.KV
	node      *maelstrom.Node
}

func NewKafkaSever(node *maelstrom.Node) *KafkaSever {
	log.SetOutput(os.Stderr)
	linKV := maelstrom.NewLinKV(node)
	seqKV := maelstrom.NewSeqKV(node)
	return &KafkaSever{
		log:   make(map[string][]Message),
		lock:  &sync.RWMutex{},
		linKV: linKV,
		seqKV: seqKV,
		node:  node,
	}
}

func (s *KafkaSever) Send(msg *SendMessage, ctx context.Context) SendMessageReply {
	key, _ := strconv.Atoi(msg.Key)
	targetNode := s.node.NodeIDs()[key%len(s.node.NodeIDs())]
	if s.node.ID() != targetNode {
		reply, err := s.node.SyncRPC(ctx, targetNode, msg)
		if err != nil {
			log.Printf("failed sending send message %v to %s", msg, targetNode)
			return msg.Reply(-1)
		}

		sendReply := new(SendMessageReply)
		if err := json.Unmarshal(reply.Body, sendReply); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return msg.Reply(-1)
		}

		return msg.Reply(sendReply.Offset)
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	messages := s.log[msg.Key]
	if messages == nil {
		messages = make([]Message, 0)
	}

	modifiedMessage := append(messages, NewMessage(len(messages), msg.Value))
	err := s.linKV.CompareAndSwap(ctx, msg.Key, messages, modifiedMessage, true)
	if err == nil {
		s.log[msg.Key] = modifiedMessage
		return msg.Reply(len(messages))
	}

	return msg.Reply(-1)
}

func (s *KafkaSever) Poll(msg *PollMessage, ctx context.Context) PollMessageReply {
	messages := make(map[string][]Message)
	s.lock.Lock()
	defer s.lock.Unlock()
	for key, offset := range msg.Offsets {
		logMessages, ok := s.log[key]
		if !ok {
			err := s.linKV.ReadInto(ctx, key, &logMessages)

			if maelstrom.ErrorCode(err) == maelstrom.KeyDoesNotExist {
				log.Printf("key: %s not found in logs", key)
				continue
			}

			if err != nil {
				log.Printf("error occurred while fetching %s", key)
				continue
			}
		}

		if len(logMessages) < offset {
			log.Printf("size of key: %s was %d which is less than offset %d", key, len(logMessages), offset)
			continue
		}

		messages[key] = logMessages[offset:]
	}

	return PollMessageReply{MessageType: "poll_ok", Messages: messages}
}

func (s *KafkaSever) CommitOffsets(msg *CommitOffsets, ctx context.Context) CommitOffsetsReply {
	for key, offset := range msg.Offsets {
		for {
			existingOffset, err := s.seqKV.ReadInt(ctx, key)
			if existingOffset > offset {
				log.Printf("existing offset %d greater than update %s:%d", existingOffset, key, offset)
				break
			}

			if err != nil || maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
				log.Printf("error while trying to read value of %s: %v", key, err)
				break
			}

			err = s.seqKV.CompareAndSwap(ctx, key, existingOffset, offset, true)
			if err == nil {
				log.Printf("offset update %s:%d", key, offset)
				break
			}

			if maelstrom.ErrorCode(err) == maelstrom.PreconditionFailed {
				continue
			}

			log.Printf("failed to update offset for %s to %d due to error %v", key, offset, err)
			break
		}
	}

	return msg.Reply()
}

func (s *KafkaSever) ListCommitedOffsets(msg *ListCommittedOffsets, ctx context.Context) ListCommittedOffsetsReply {
	offsets := make(map[string]int)
	for _, key := range msg.Keys {
		offset, err := s.seqKV.ReadInt(ctx, key)
		if maelstrom.ErrorCode(err) == maelstrom.KeyDoesNotExist {
			log.Printf("log offset not found of key:%s", key)
			continue
		}

		if err != nil {
			log.Printf("failed to fetch key: %s", key)
			continue
		}

		offsets[key] = offset
	}

	return ListCommittedOffsetsReply{MessageType: "list_committed_offsets_ok", Offsets: offsets}
}

package growonlycounter

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const GROW_ONLY_KEY string = "groww"

type GrowOnlyCounterServer struct {
	n  *maelstrom.Node
	kv *maelstrom.KV
}

func NewGrowOnlyCounterServer(n *maelstrom.Node) GrowOnlyCounterServer {
	log.SetOutput(os.Stderr)
	kv := maelstrom.NewSeqKV(n)
	return GrowOnlyCounterServer{
		n:  n,
		kv: kv,
	}
}

func (s *GrowOnlyCounterServer) Read(msg *ReadMessage, ctx context.Context) ReadMessageReply {
	s.kv.Write(ctx, strconv.Itoa(int(rand.Int63())), nil)
	value, err := s.kv.ReadInt(ctx, GROW_ONLY_KEY)
	if err != nil {
		log.Printf("failed to read %s from sequential KV\n", GROW_ONLY_KEY)
		return ReadMessageReply{}
	}
	return msg.Reply(value)
}

func (s *GrowOnlyCounterServer) Add(msg *AddMessage, ctx context.Context) AddMessageReply {
	for {
		value, err := s.kv.ReadInt(ctx, GROW_ONLY_KEY)
		if err != nil && maelstrom.ErrorCode(err) != maelstrom.KeyDoesNotExist {
			log.Printf("failed to read %s from sequential KV", GROW_ONLY_KEY)
			continue
		}

		log.Printf("read value %d\n", value)
		err = s.kv.CompareAndSwap(ctx, GROW_ONLY_KEY, value, value+msg.Delta, true)
		if err == nil {
			log.Printf("value updated to %d\n", value+msg.Delta)
			return msg.Reply()
		}

		log.Printf("retrying with delta %d\n", msg.Delta)
	}
}

package uniqueidgeneration

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type UniqueIdServer struct {
	n       *maelstrom.Node
	counter uint16
	mutex   *sync.Mutex
}

func NewUniqueIdServer(n *maelstrom.Node) UniqueIdServer {
	return UniqueIdServer{
		n:       n,
		counter: 0,
		mutex:   &sync.Mutex{},
	}
}

func (s *UniqueIdServer) HandleMessage(m maelstrom.Message) error {
	receivedMessage := new(UniqueIdMessage)
	if err := json.Unmarshal(m.Body, receivedMessage); err != nil {
		return err
	}

	uniqueId, err := s.GenerateUniqueId(m.Src, m.Dest)
	if err != nil {
		return err
	}

	return s.n.Reply(m, receivedMessage.Reply(uniqueId))
}

func (s *UniqueIdServer) GenerateUniqueId(src string, dest string) (uint64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.counter += 1
	epoch := uint64(time.Now().Unix())

	srcInt, err := strconv.Atoi(strings.Trim(src, "c"))
	if err != nil {
		return 0, err
	}
	destInt, err := strconv.Atoi(strings.Trim(dest, "n"))
	if err != nil {
		return 0, err
	}

	return epoch<<32 | uint64(srcInt)<<24 | uint64(destInt)<<16 | uint64(s.counter), nil
}

package broadcast

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastServer struct {
	n              *maelstrom.Node
	messages       *sync.Map
	neighbours     []string
	passingChannel chan struct {
		int
		string
	}
	gossipTicker *time.Ticker
}

func NewBroadcastServer(n *maelstrom.Node, tickDuration time.Duration) BroadcastServer {
	log.SetOutput(os.Stderr)
	return BroadcastServer{
		n:          n,
		messages:   &sync.Map{},
		neighbours: make([]string, 0),
		passingChannel: make(chan struct {
			int
			string
		}, 200),
		gossipTicker: time.NewTicker(tickDuration),
	}
}

func (s *BroadcastServer) getMessages() []int {
	stored_messages := make([]int, 0)
	s.messages.Range(func(key, value any) bool {
		if intKey, ok := key.(int); ok {
			stored_messages = append(stored_messages, intKey)
		}
		return true
	})
	return stored_messages
}

func (s *BroadcastServer) Read(msg *ReadMessage) ReadMessageReply {
	return msg.Reply(s.getMessages())
}

func (s *BroadcastServer) Broadcast(msg *BroadcastMessage, src string) (BroadcastMessageReply, bool) {
	replyBack := true
	if msg.MessageID == 0 {
		replyBack = false
	}

	if _, found := s.messages.LoadOrStore(msg.Message, struct{}{}); found {
		log.Printf("message already present %d", msg.Message)
		return msg.Reply(), replyBack
	}

	log.Printf("%s: broadcasting message %d", s.n.ID(), msg.Message)
	s.passingChannel <- struct {
		int
		string
	}{msg.Message, src}

	return msg.Reply(), replyBack
}

func (s *BroadcastServer) getNeighbours() []string {
	nodeIDStr := s.n.ID()
	if !strings.HasPrefix(nodeIDStr, "n") {
		log.Printf("Error: Node ID %s does not have 'n' prefix", nodeIDStr)
		return []string{}
	}

	idPart := strings.TrimPrefix(nodeIDStr, "n")
	nodeNum, err := strconv.Atoi(idPart)
	if err != nil {
		log.Printf("Error converting node ID %s to int: %v", idPart, err)
		return []string{}
	}

	totalNodes := len(s.n.NodeIDs())
	child1 := (2 * nodeNum) % totalNodes
	child2 := (2*nodeNum + 1) % totalNodes
	neighbours := []string{"n" + strconv.Itoa(child1), "n" + strconv.Itoa(child2)}
	log.Printf("Node %s, calculated neighbours: %v", nodeIDStr, neighbours)
	return neighbours
}

func (s *BroadcastServer) Topology(msg *TopologyMessage) TopologyMessageReply {
	s.neighbours = s.getNeighbours()
	return msg.Reply()
}

func (s *BroadcastServer) Gossip(messages []int, source string) {
	for _, message := range messages {
		if _, ok := s.messages.LoadOrStore(message, struct{}{}); ok {
			continue
		}

		log.Printf("%s: found new message %d from %s", s.n.ID(), message, source)
		s.passingChannel <- struct {
			int
			string
		}{message, source}
	}
}

func (s *BroadcastServer) getRandomNodes(count int) []string {
	nodes := s.n.NodeIDs()
	rand.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	if count > len(nodes) {
		return nodes
	}

	return nodes[:count]
}

func (s *BroadcastServer) Gossiper(ctx context.Context, randomNodes int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.gossipTicker.C:
			{
				randomNodes := s.getRandomNodes(randomNodes)
				messages := s.getMessages()
				for _, randomNode := range randomNodes {
					if randomNode == s.n.ID() {
						continue
					}

					s.n.Send(randomNode, GossipMessage{MessageType: "gossip", Messages: messages})
				}
			}
		}
	}
}

func (s *BroadcastServer) SendToNeighbours(context context.Context) {
	for message := range s.passingChannel {
		wg := &sync.WaitGroup{}
		for _, node := range s.neighbours {
			if node == message.string {
				continue
			}

			wg.Add(1)
			go func(node string, wg *sync.WaitGroup) {
				s.n.Send(node, BroadcastMessage{MessageType: "broadcast", Message: message.int})
				log.Printf("%s: sent %d to %s", s.n.ID(), message.int, node)
				wg.Done()
			}(node, wg)
		}
		wg.Wait()
	}
}

func (s *BroadcastServer) Stop() {
	close(s.passingChannel)
	s.gossipTicker.Stop()
}

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
	gossipTickDuration     time.Duration
	neighboursTickDuration time.Duration
}

func NewBroadcastServer(n *maelstrom.Node, gossipTickDuration time.Duration, neighboursTickDuration time.Duration) BroadcastServer {
	log.SetOutput(os.Stderr)
	return BroadcastServer{
		n:          n,
		messages:   &sync.Map{},
		neighbours: make([]string, 0),
		passingChannel: make(chan struct {
			int
			string
		}, 200),
		gossipTickDuration:     gossipTickDuration,
		neighboursTickDuration: neighboursTickDuration,
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

	totalNodes := len(s.n.NodeIDs()) + 1
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
	gossipTicker := time.NewTicker(s.gossipTickDuration)
	defer gossipTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			break
		case <-gossipTicker.C:
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
	neighboursTicker := time.NewTicker(s.neighboursTickDuration)
	defer neighboursTicker.Stop()

	messageBatch := make([]int, 0, 10)
	for {
		select {
		case <-context.Done():
			break
		case message, ok := <-s.passingChannel:
			if !ok {
				break
			}
			messageBatch = append(messageBatch, message.int)
			log.Printf("%s: received message %d from %s", s.n.ID(), message.int, message.string)
		case <-neighboursTicker.C:
			wg := &sync.WaitGroup{}
			for _, node := range s.neighbours {
				wg.Add(1)
				go func(node string, wg *sync.WaitGroup, messages []int) {
					s.n.Send(node, GossipMessage{MessageType: "gossip", Messages: messages})
					log.Printf("%s: sent a total of %d messages to %s", s.n.ID(), len(messages), node)
					wg.Done()
				}(node, wg, messageBatch)
				wg.Wait()
			}
			messageBatch = make([]int, 0, 10)
		}
	}
}

func (s *BroadcastServer) Stop() {
	close(s.passingChannel)
}

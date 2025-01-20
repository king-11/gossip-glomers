package broadcast

import (
	"math/rand"
	"slices"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastServer struct {
	n *maelstrom.Node
	messages []int
	neighbours []string
	passingChannel chan int
	gossipTicker *time.Ticker
}

func NewBroadcastServer(n *maelstrom.Node) BroadcastServer {
	return BroadcastServer{
		n: n,
		messages: make([]int, 0, 10),
		neighbours: make([]string, 0, 5),
		passingChannel: make(chan int, 200),
		gossipTicker: time.NewTicker(time.Second * 5),
	}
}

func (s *BroadcastServer) Read(msg *ReadMessage) ReadMessageReply {
	return msg.Reply(s.messages)
}

func (s *BroadcastServer) Broadcast(msg *BroadcastMessage, src string) (BroadcastMessageReply, bool) {
	replyBack := true
	if msg.MessageID == 0 {
		replyBack = false
	}

	if slices.Index(s.messages, msg.Message) != -1 {
		return msg.Reply(), replyBack
	}

	s.messages = append(s.messages, msg.Message)
	s.passingChannel <- msg.Message

	return msg.Reply(), replyBack
}

func (s *BroadcastServer) sendAllMessagesToDestination(destination string) {
	for _, message := range s.messages {
		s.n.Send(destination, BroadcastMessage { MessageType: "broadcast", Message: message })
	}
}

func (s *BroadcastServer) Gossiper() {
	nonNeighbours := make([]string, 0, len(s.n.NodeIDs()))
	for {
		if len(s.neighbours) != 0 {
			break
		}

		time.Sleep(time.Second * 1)
	}

	for _, node := range s.n.NodeIDs() {
		if slices.Index(s.neighbours, node) == -1 {
			nonNeighbours = append(nonNeighbours, node)
		}
	}

	if len(nonNeighbours) == 0 {
		return
	}

	for range s.gossipTicker.C {
		randomNode := nonNeighbours[rand.Intn(len(nonNeighbours))]
		s.sendAllMessagesToDestination(randomNode)
	}
}

func (s *BroadcastServer) SendToNeighbours() {
	for message := range s.passingChannel {
		wg := &sync.WaitGroup{}
		for _, node := range s.neighbours {
			wg.Add(1)
			go func(node string, wg *sync.WaitGroup) {
				s.n.Send(node, BroadcastMessage { MessageType: "broadcast", Message: message})
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

func (s *BroadcastServer) Topology(msg *TopologyMessage) TopologyMessageReply {
	topology := msg.Topology
	if val, contains := topology[s.n.ID()]; contains {
		s.neighbours = val
	}

	return msg.Reply()
}

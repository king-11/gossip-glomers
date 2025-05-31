package broadcast

import (
	"testing"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func TestRead(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewBroadcastServer(n, time.Second, time.Second)
	server.messages.Store(20, struct{}{})
	readMessage := ReadMessage{MessageType: "read"}

	reply := server.Read(&readMessage)

	if reply.MessageType != "read_ok" || len(reply.Messages) != 1 || reply.Messages[0] != 20 {
		t.FailNow()
	}
}

func TestBroadcastNewMessage(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewBroadcastServer(n, time.Second, time.Second)
	broadcastMessage := BroadcastMessage{MessageType: "broadcast", Message: 1000, MessageID: 1}

	broadcastReply, replyBack := server.Broadcast(&broadcastMessage, "c2")

	if replyBack == false {
		t.Fail()
	}

	if broadcastReply.MessageType != "broadcast_ok" {
		t.Fail()
	}

	if _, ok := server.messages.Load(1000); !ok {
		t.Fail()
	}

	select {
	case message := <-server.passingChannel:
		if message.int != 1000 || message.string != "c2" {
			t.Fail()
		}
	case <-time.After(time.Millisecond * 5):
		t.Fail()
	}
}

func TestBroadcastDuplicateMessage(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewBroadcastServer(n, time.Second, time.Second)
	server.messages.Store(1000, struct{}{})
	broadcastMessage := BroadcastMessage{MessageType: "broadcast", Message: 1000}

	broadcastReply, replyBack := server.Broadcast(&broadcastMessage, "n2")

	if replyBack == true {
		t.Fail()
	}

	if broadcastReply.MessageType != "broadcast_ok" {
		t.Fail()
	}

	select {
	case <-server.passingChannel:
		t.Fail()
	case <-time.After(time.Millisecond * 2):
	}
}

func TestTopology(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewBroadcastServer(n, time.Second, time.Second)
	topologyMessage := TopologyMessage{MessageType: "topology", Topology: map[string][]string{"n1": {"n2", "n3"}, "": {"n1"}}}

	toplogyReply := server.Topology(&topologyMessage)

	if toplogyReply.MessageType != "topology_ok" {
		t.FailNow()
	}
}

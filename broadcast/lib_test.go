package broadcast

import (
	"testing"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func TestBroadcast(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewBroadcastServer(n)

	readMessage := ReadMessage{MessageType: "read"}
	reply := server.Read(&readMessage)

	if reply.MessageType != "read_ok" || len(reply.Messages) != 0 {
		t.FailNow()
	}

	broadcastMessage := BroadcastMessage{MessageType: "broadcast", Message: 1000}
	broadcastReply, _ := server.Broadcast(&broadcastMessage, "c1")
	if broadcastReply.MessageType != "broadcast_ok" {
		t.FailNow()
	}

	reply = server.Read(&readMessage)
	if len(reply.Messages) == 0 || reply.Messages[0] != 1000 {
		t.FailNow()
	}

	topologyMessage := TopologyMessage{MessageType: "topology", Topology: map[string][]string{"n1": {"n2", "n3"}, "": {"n1"}}}
	toplogyReply := server.Topology(&topologyMessage)
	if toplogyReply.MessageType != "topology_ok" {
		t.FailNow()
	}
}

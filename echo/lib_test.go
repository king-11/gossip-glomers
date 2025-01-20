package echo

import "testing"

func TestReply(t *testing.T) {
	message := EchoMessage{
		MessageType: "echo",
		MessageId: 1,
		Echo: "echo hello",
	}

	reply := message.Reply()
	if message.Echo != reply.Echo || message.MessageId != reply.MessageId || reply.InReplyTo != message.MessageId || reply.MessageType != "echo_ok" {
		t.Fatalf("unexpected reply %v", reply)
	}
}

package echo

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type EchoMessage struct {
	MessageType string `json:"type"`
	MessageId int `json:"msg_id"`
	Echo string `json:"echo"`
}

type EchoMessageReply struct {
	MessageType string `json:"type"`
	MessageId int	`json:"msg_id"`
	InReplyTo int `json:"in_reply_to"`
	Echo string `json:"echo"`
}

func (m *EchoMessage) Reply() EchoMessageReply {
	return EchoMessageReply {
		MessageType: "echo_ok",
		MessageId: m.MessageId,
		InReplyTo: m.MessageId,
		Echo: m.Echo,
	}
}

func HandleEcho(msg maelstrom.Message, n *maelstrom.Node) error {
	body := new(EchoMessage)
	if err := json.Unmarshal(msg.Body, body); err != nil {
		return err
	}

	reply := body.Reply()
	return n.Reply(msg, reply)
}

package growonlycounter

type AddMessage struct {
	MessageType string `json:"type"`
	Delta       int    `json:"delta"`
}

func (m *AddMessage) Reply() AddMessageReply {
	return AddMessageReply{MessageType: "add_ok"}
}

type AddMessageReply struct {
	MessageType string `json:"type"`
}

type ReadMessage struct {
	MessageType string `json:"type"`
}

type ReadMessageReply struct {
	MessageType string `json:"type"`
	Value       int    `json:"value"`
}

func (m *ReadMessage) Reply(value int) ReadMessageReply {
	return ReadMessageReply{MessageType: "read_ok", Value: value}
}

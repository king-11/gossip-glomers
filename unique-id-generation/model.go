package uniqueidgeneration

type UniqueIdMessage struct {
	MessageType string `json:"type"`
}

type UniqueIdMessageReply struct {
	MessageType string `json:"type"`
	Id          string `json:"id"`
}

func (m *UniqueIdMessage) Reply(id string) UniqueIdMessageReply {
	return UniqueIdMessageReply{
		MessageType: "generate_ok",
		Id:          id,
	}
}

package uniqueidgeneration

type UniqueIdMessage struct {
	MessageType string `json:"type"`
}

type UniqueIdMessageReply struct {
	MessageType string `json:"type"`
	Id uint64 `json:"id"`
}

func (m *UniqueIdMessage) Reply(id uint64) UniqueIdMessageReply {
	return UniqueIdMessageReply{
		MessageType: "generate_ok",
		Id: id,
	}
}

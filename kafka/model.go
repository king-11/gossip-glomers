package kafka

type SendMessage struct {
	MessageType string `json:"type"`
	Key         string `json:"key"`
	Value       int    `json:"msg"`
}

type SendMessageReply struct {
	MessageType string `json:"type"`
	Offset      int    `json:"offset"`
}

func (s *SendMessage) Reply(offset int) SendMessageReply {
	return SendMessageReply{
		MessageType: "send_ok",
		Offset:      offset,
	}
}

type Offsets map[string]int

type Message [2]int

func NewMessage(offset int, value int) Message {
	return [2]int{offset, value}
}

func (m *Message) offset() int {
	return m[0]
}

func (m *Message) value() int {
	return m[1]
}

type PollMessage struct {
	MessageType string  `json:"type"`
	Offsets     Offsets `json:"offsets"`
}

type PollMessageReply struct {
	MessageType string               `json:"type"`
	Messages    map[string][]Message `json:"msgs"`
}

type CommitOffsets struct {
	MessageType string  `json:"type"`
	Offsets     Offsets `json:"offsets"`
}

type CommitOffsetsReply struct {
	MessageType string `json:"type"`
}

func (co *CommitOffsets) Reply() CommitOffsetsReply {
	return CommitOffsetsReply{MessageType: "commit_offsets_ok"}
}

type ListCommittedOffsets struct {
	MessageType string   `json:"type"`
	Keys        []string `json:"keys"`
}

type ListCommittedOffsetsReply struct {
	MessageType string  `json:"type"`
	Offsets     Offsets `json:"offsets"`
}

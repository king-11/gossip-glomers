package broadcast

type ReadMessage struct {
	MessageType string `json:"type"`
}

type ReadMessageReply struct {
	MessageType string `json:"type"`
	Messages    []int  `json:"messages"`
}

func (m *ReadMessage) Reply(messages []int) ReadMessageReply {
	return ReadMessageReply{
		MessageType: "read_ok",
		Messages:    messages,
	}
}

type TopologyMessage struct {
	MessageType string              `json:"type"`
	Topology    map[string][]string `json:"topology"`
}

type TopologyMessageReply struct {
	MessageType string `json:"type"`
}

func (m *TopologyMessage) Reply() TopologyMessageReply {
	return TopologyMessageReply{
		MessageType: "topology_ok",
	}
}

type BroadcastMessage struct {
	MessageType string `json:"type"`
	Message     int    `json:"message"`
	MessageID   int    `json:"msg_id"`
}

type BroadcastMessageReply struct {
	MessageType string `json:"type"`
}

func (m *BroadcastMessage) Reply() BroadcastMessageReply {
	return BroadcastMessageReply{
		MessageType: "broadcast_ok",
	}
}

type GossipMessage struct {
	MessageType string `json:"type"`
	Messages    []int  `json:"messages"`
	MessageID   int    `json:"msg_id"`
}

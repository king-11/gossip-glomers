package kafka

import (
	"maps"
	"slices"
	"testing"
)

func TestKafkaSend(t *testing.T) {
	kafkaServer := NewKafkaSever()
	sendMessage := SendMessage{MessageType: "send", Value: 11, Key: "luck"}

	reply := kafkaServer.Send(&sendMessage)

	if reply.MessageType != "send_ok" {
		t.Errorf("expected reply of type 'send_ok' but was %s", reply.MessageType)
	}

	if reply.Offset != 0 {
		t.Errorf("initial offset should be 0 but was %d", reply.Offset)
	}

	if messages, ok := kafkaServer.log["luck"]; !ok {
		t.Error("key 'luck' not found in server logs")
	} else {
		if len(messages) != 1 {
			t.Error("sent message not present in the logs")
		}
		if messages[0][0] != 0 || messages[0][1] != 11 {
			t.Error("sent message not found in server (0,11)")
		}
	}
}

func TestKafkaPoll(t *testing.T) {
	kafkaServer := NewKafkaSever()
	kafkaServer.log["luck"] = []Message{NewMessage(0, 11), NewMessage(1, 12), NewMessage(2, 45)}
	kafkaServer.log["prize"] = []Message{NewMessage(0, 1), NewMessage(1, 4)}
	pollMessage := PollMessage{MessageType: "poll", Offsets: Offsets{"luck": 0, "prize": 1}}

	reply := kafkaServer.Poll(&pollMessage)

	if reply.MessageType != "poll_ok" {
		t.Errorf("incorrect message type of %s expected 'poll_ok'", reply.MessageType)
	}

	if !slices.Equal(reply.Messages["luck"], []Message{NewMessage(0, 11), NewMessage(1, 12), NewMessage(2, 45)}) {
		t.Errorf("messages of key 'luck' not as expected %v", reply.Messages["luck"])
	}

	if !slices.Equal(reply.Messages["prize"], []Message{NewMessage(1, 4)}) {
		t.Errorf("messages of key 'prize' not as expected %v", reply.Messages["prize"])
	}
}

func TestKafkaCommitOffsets(t *testing.T) {
	kafkaServer := NewKafkaSever()
	kafkaServer.log["luck"] = []Message{NewMessage(0, 11), NewMessage(1, 12), NewMessage(2, 45)}
	kafkaServer.log["prize"] = []Message{NewMessage(0, 1), NewMessage(1, 4)}
	commitOffsetsMessage := CommitOffsets{MessageType: "commit_offsets", Offsets: Offsets{
		"luck":  2,
		"prize": 1,
	}}

	reply := kafkaServer.CommitOffsets(&commitOffsetsMessage)

	if reply.MessageType != "commit_offsets_ok" {
		t.Errorf("expected message of type 'commit_offsets_ok' but was %s", reply.MessageType)
	}

	if kafkaServer.logOffset["luck"] != 2 || kafkaServer.logOffset["prize"] != 1 {
		t.Errorf("luck:%d expected:2, prize:%d expected:1", kafkaServer.logOffset["luck"], kafkaServer.logOffset["prize"])
	}
}

func TestListCommittedOffsets(t *testing.T) {
	kafkaServer := NewKafkaSever()
	kafkaServer.log["luck"] = []Message{NewMessage(0, 11), NewMessage(1, 12), NewMessage(2, 45)}
	kafkaServer.log["prize"] = []Message{NewMessage(0, 1), NewMessage(1, 4)}
	kafkaServer.logOffset["luck"] = 2
	kafkaServer.logOffset["prize"] = 1
	listCommittedOffsets := ListCommittedOffsets{MessageType: "list_committed_offsets", Keys: []string{"luck", "nozzle", "prize"}}

	reply := kafkaServer.ListCommitedOffsets(&listCommittedOffsets)

	if reply.MessageType != "list_committed_offsets_ok" {
		t.Errorf("expected message of type 'commit_offsets_ok' but was %s", reply.MessageType)
	}

	if !maps.Equal(reply.Offsets, Offsets{"luck": 2, "prize": 1}) {
		t.Errorf("expected values map[luck:2, prize:1] but found %v", reply.Offsets)
	}
}

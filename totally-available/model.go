package totallyavailable

import "time"

type Operation = [3]any

func OperationResult(kind string, key int, value any) Operation {
	return Operation{kind, key, value}
}

func IsRead(o Operation) bool {
	op := o[0].(string)

	return op == "r"
}

func IsWrite(o Operation) bool {
	return !IsRead(o)
}

func GetKey(o Operation) int {
	return int(o[1].(float64))
}

func GetValue(o Operation) int {
	return int(o[2].(float64))
}

type TxnRequest struct {
	Type       string      `json:"type"`
	MessageId  uint        `json:"msg_id"`
	Operations []Operation `json:"txn"`
}

func (t *TxnRequest) Reply(operationResults []Operation) TxnReply {
	return TxnReply{
		Type:       "txn_ok",
		MessageId:  t.MessageId,
		InReplyTo:  t.MessageId,
		Operations: operationResults,
	}
}

type TxnReply struct {
	Type       string      `json:"type"`
	MessageId  uint        `json:"msg_id"`
	InReplyTo  uint        `json:"in_reply_to"`
	Operations []Operation `json:"txn"`
}

type WriteMessage struct {
	MessageType string            `json:"type"`
	Requests    []WriteKeyRequest `json:"requests"`
}

type WriteKeyRequest struct {
	Key       int  `json:"key"`
	Value     int  `json:"value"`
	Timestamp uint `json:"timestamp"`
}

func NewWriteKeyRequest(key int, value int) WriteKeyRequest {
	return WriteKeyRequest{
		Key:       key,
		Value:     value,
		Timestamp: uint(time.Now().Unix()),
	}
}

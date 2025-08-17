package totallyavailable

import (
	"log"
	"os"
	"sort"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type TotallyAvailableNode struct {
	keyLocks *sync.Map
	kv       *sync.Map
}

func NewTotallyAvailableNode(n *maelstrom.Node) TotallyAvailableNode {
	log.SetOutput(os.Stderr)
	return TotallyAvailableNode{
		keyLocks: &sync.Map{},
		kv:       &sync.Map{},
	}
}

func (ta *TotallyAvailableNode) lockForKey(key int) *sync.RWMutex {
	unknownLock, _ := ta.keyLocks.LoadOrStore(key, &sync.RWMutex{})
	lock := unknownLock.(*sync.RWMutex)
	return lock
}

func (ta *TotallyAvailableNode) performWrite(key int, value int) Operation {
	ta.kv.Store(key, value)

	log.Printf("key:%d set to %d", key, value)
	return OperationResult("w", key, value)
}

func (ta *TotallyAvailableNode) performRead(key int) Operation {
	value, ok := ta.kv.Load(key)
	if !ok {
		log.Printf("key:%d not found", key)
		return OperationResult("r", key, nil)
	}

	log.Printf("key:%d has value %d", key, value)
	return OperationResult("r", key, value)
}

func (ta *TotallyAvailableNode) lockKeys(writeKeys []int) []*sync.RWMutex {
	locked := make([]*sync.RWMutex, 0, len(writeKeys))
	for _, k := range writeKeys {
		l := ta.lockForKey(k)
		l.Lock()
		locked = append(locked, l)
		log.Printf("acquired write lock for key:%d", k)
	}

	return locked
}

func (ta *TotallyAvailableNode) unlockLocks(locks []*sync.RWMutex) {
	for _, lock := range locks {
		lock.Unlock()
	}
}

func (ta *TotallyAvailableNode) Transaction(msg *TxnRequest) TxnReply {
	writeKeysSet := make(map[int]struct{})
	for _, op := range msg.Operations {
		if IsWrite(op) {
			writeKeysSet[GetKey(op)] = struct{}{}
		}
	}

	writeKeys := make([]int, 0, len(writeKeysSet))
	for k := range writeKeysSet {
		writeKeys = append(writeKeys, k)
	}
	sort.Ints(writeKeys)

	locks := ta.lockKeys(writeKeys)
	defer ta.unlockLocks(locks)

	results := make([]Operation, len(msg.Operations))
	for idx, op := range msg.Operations {
		key := GetKey(op)
		isWrite := IsWrite(op)

		if isWrite {
			results[idx] = ta.performWrite(key, GetValue(op))
		} else {
			results[idx] = ta.performRead(key)
		}
	}

	return msg.Reply(results)
}

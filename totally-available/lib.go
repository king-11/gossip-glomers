package totallyavailable

import (
	"context"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	TICKER_TIME           = 1 * time.Second
	MAXIMUM_STORED_WRITES = 25
)

type TotallyAvailableNode struct {
	keyLocks       *sync.Map
	kv             *sync.Map
	lastWrite      *sync.Map
	node           *maelstrom.Node
	requestChannel chan WriteKeyRequest
}

func NewTotallyAvailableNode(n *maelstrom.Node) TotallyAvailableNode {
	log.SetOutput(os.Stderr)
	return TotallyAvailableNode{
		keyLocks:       &sync.Map{},
		kv:             &sync.Map{},
		lastWrite:      &sync.Map{},
		node:           n,
		requestChannel: make(chan WriteKeyRequest, MAXIMUM_STORED_WRITES*2),
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

func (ta *TotallyAvailableNode) lockKeys(keys []int, isRead bool) []*sync.RWMutex {
	slices.Sort(keys)
	keys = slices.Compact(keys)
	locked := make([]*sync.RWMutex, 0, len(keys))
	for _, k := range keys {
		l := ta.lockForKey(k)
		if isRead {
			l.RLock()
		} else {
			l.Lock()
		}

		locked = append(locked, l)
		log.Printf("acquired lock for key:%d", k)
	}

	return locked
}

func (ta *TotallyAvailableNode) unlockLocks(locks []*sync.RWMutex, keys []int, areReadLocks bool) {
	for idx, lock := range locks {
		if areReadLocks {
			lock.RUnlock()
		} else {
			lock.Unlock()
		}

		log.Printf("key:%d unlocked", keys[idx])
	}
}

func (ta *TotallyAvailableNode) Transaction(msg *TxnRequest) TxnReply {
	writeKeysSet := make(map[int]struct{})
	readKeysSet := make(map[int]struct{})
	for _, op := range msg.Operations {
		if IsWrite(op) {
			writeKeysSet[GetKey(op)] = struct{}{}
		}
	}

	for _, op := range msg.Operations {
		key := GetKey(op)
		if _, ok := writeKeysSet[key]; ok {
			continue
		}

		readKeysSet[key] = struct{}{}
	}

	readKeys := make([]int, 0, len(readKeysSet))
	for k := range readKeysSet {
		readKeys = append(readKeys, k)
	}
	readLocks := ta.lockKeys(readKeys, true)
	defer ta.unlockLocks(readLocks, readKeys, true)

	writeKeys := make([]int, 0, len(writeKeysSet))
	for k := range writeKeysSet {
		writeKeys = append(writeKeys, k)
	}
	writeLocks := ta.lockKeys(writeKeys, false)
	defer ta.unlockLocks(writeLocks, writeKeys, false)

	results := make([]Operation, len(msg.Operations))
	for idx, op := range msg.Operations {
		key := GetKey(op)
		isWrite := IsWrite(op)

		if isWrite {
			value := GetValue(op)
			results[idx] = ta.performWrite(key, GetValue(op))
			ta.requestChannel <- NewWriteKeyRequest(key, value)
		} else {
			results[idx] = ta.performRead(key)
		}
	}

	return msg.Reply(results)
}

func (ta *TotallyAvailableNode) WriteServer(ctx context.Context) {
	ticker := time.NewTicker(TICKER_TIME)
	storedWrites := make([]WriteKeyRequest, 0, MAXIMUM_STORED_WRITES)
	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			if len(storedWrites) < MAXIMUM_STORED_WRITES || len(storedWrites) == 0 {
				continue
			}

			writeMessage := WriteMessage{MessageType: "write", Requests: storedWrites}
			wg := &sync.WaitGroup{}
			for _, dest := range ta.node.NodeIDs() {
				if dest == ta.node.ID() {
					continue
				}

				wg.Add(1)
				go func(dest string, wg *sync.WaitGroup) {
					ta.node.Send(dest, writeMessage)
					log.Printf("sent to %s: %v", dest, writeMessage)
					wg.Done()
				}(dest, wg)
			}
			wg.Wait()

			storedWrites = make([]WriteKeyRequest, 0, MAXIMUM_STORED_WRITES)
		case writeRequest, open := <-ta.requestChannel:
			if !open {
				break
			}

			log.Printf("received new write request %v", writeRequest)
			storedWrites = append(storedWrites, writeRequest)
		}
	}
}

func (ta *TotallyAvailableNode) Write(requests []WriteKeyRequest) {
	writeKeys := make([]int, len(requests))
	for idx, req := range requests {
		writeKeys[idx] = req.Key
	}

	locks := ta.lockKeys(writeKeys, false)
	defer ta.unlockLocks(locks, writeKeys, false)

	for _, req := range requests {
		lastWrite, loaded := ta.lastWrite.LoadOrStore(req.Key, req.Timestamp)
		if !loaded && lastWrite.(uint) > req.Timestamp {
			log.Printf("key:%d not updated as %d > %d", req.Key, lastWrite.(uint), req.Timestamp)
			continue
		}

		ta.performWrite(req.Key, req.Value)
	}
}

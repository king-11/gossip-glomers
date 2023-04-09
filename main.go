package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "echo_ok"

		return n.Reply(msg, body)
	})

	uniqueID := 1
	n.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "generate_ok"

		id := fmt.Sprintf("%d-%s-%s-%d", time.Now().Unix(), msg.Src, msg.Dest, uniqueID)
		uniqueID += 1

		body["id"] = id

		return n.Reply(msg, body)
	})

	seen := make(map[int]struct{})
	n.Handle("broadcast", func(msg maelstrom.Message) error {
		type Body struct {
			TypeOf  string `json:"type"`
			Message int    `json:"message"`
		}

		var body Body
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		seen[body.Message] = struct{}{}

		resp_body := make(map[string]any)
		resp_body["type"] = "broadcast_ok"

		return n.Reply(msg, resp_body)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		body := make(map[string]any)
		vals := make([]int, 0)

		for val := range seen {
			vals = append(vals, val)
		}

		body["type"] = "read_ok"
		body["messages"] = vals

		return n.Reply(msg, body)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		body := make(map[string]any)

		body["type"] = "topology_ok"

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

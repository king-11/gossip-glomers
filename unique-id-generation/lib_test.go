package uniqueidgeneration

import (
	"testing"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func TestUniqueId(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewUniqueIdServer(n)

	_, err := server.GenerateUniqueId("c1", "n2")
	if err != nil {
		t.Fatal(err)
	}
}

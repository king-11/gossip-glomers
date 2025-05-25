//go:build goexperiment.synctest

package uniqueidgeneration

import (
	"testing"
	"testing/synctest"

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

func TestGenerateTwoDifferentUniqueIds(t *testing.T) {
	n := maelstrom.NewNode()
	server := NewUniqueIdServer(n)

	uniqueId1, err := server.GenerateUniqueId("c1", "n2")
	if err != nil {
		t.Fatal(err)
	}

	uniqueId2, err := server.GenerateUniqueId("c1", "n2")
	if err != nil {
		t.Fatal(err)
	}

	if uniqueId1 == uniqueId2 {
		t.Fatal("Generated two identical unique IDs, but expected them to be different")
	}
}

func TestGenerateUniqueIdsAtSameTime(t *testing.T) {
	synctest.Run(func() {
		n := maelstrom.NewNode()
		server := NewUniqueIdServer(n)
		uniqueId1, err := server.GenerateUniqueId("c1", "n2")
		if err != nil {
			t.Fail()
		}
		uniqueId2, err := server.GenerateUniqueId("c1", "n2")
		if err != nil {
			t.Fail()
		}

		if uniqueId1 == uniqueId2 {
			t.Fatal("Generated two identical unique IDs, but expected them to be different")
		}
	})
}

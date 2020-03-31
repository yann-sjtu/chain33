package p2pstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/p2pnext/protocol/types"

	"github.com/33cn/chain33/queue"
	"github.com/libp2p/go-libp2p-core/crypto"
)

func TestInit(t *testing.T) {
	q := queue.New("test")
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := priv.Bytes()
	fmt.Println(hex.EncodeToString(b))
	env := types.P2PEnv{
		QueueClient: q.Client(),
		Host:        nil,
		Discovery:   nil,
	}
	_ = env
}

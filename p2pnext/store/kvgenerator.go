package store

import (
	"encoding/hex"
	"fmt"

	"github.com/33cn/chain33/types"
	"github.com/gogo/protobuf/proto"
)

func MakeBlockHashAsKey(hash []byte) string {
	return fmt.Sprintf("/%s/%s", DhtStoreNamespace, hex.EncodeToString(hash))
}

func MakeBlockAsValue(block *types.Block) ([]byte, error) {
	return proto.Marshal(block)
}

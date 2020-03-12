package types

import (
	"errors"

	"github.com/33cn/chain33/types"
)

//按区间查询区块
type FetchBlocks struct {
	Hash   string
	Start  int64
	End    int64
	Filter bool
}

type PutPackage struct {
	Hash   string
	Blocks []*types.Block
}

var (
	ErrParams = errors.New("wrong parameter")
)

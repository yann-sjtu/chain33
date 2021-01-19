package db

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/tecbot/gorocksdb"
)

var rocksLog = log15.New("module", "db.rocksdb")

// PebbleDB db
type RocksDB struct {
	BaseDB
	db *gorocksdb.DB
}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewRocksDB(name, dir, cache)
	}
	registerDBCreator(rocksDBBackendStr, dbCreator, false)
}

// NewPebbleDB new
func NewRocksDB(name string, dir string, cache int) (*RocksDB, error) {
	dbPath := path.Join(dir, name+".db")
	_, err := os.Lstat(dbPath)
	if err != nil {
		_ = os.MkdirAll(dbPath, os.ModePerm)
	}
	options := gorocksdb.NewDefaultOptions()
	options.EnableStatistics()
	options.SetMaxWriteBufferNumber(3)
	options.SetMaxBackgroundCompactions(10)
	options.SetHashSkipListRep(2000000, 4, 4)

	blockBasedTableOptions := gorocksdb.NewDefaultBlockBasedTableOptions()
	blockBasedTableOptions.SetBlockCache(gorocksdb.NewLRUCache(64 * 1024))
	blockBasedTableOptions.SetFilterPolicy(gorocksdb.NewBloomFilter(10))
	blockBasedTableOptions.SetBlockSizeDeviation(5)
	blockBasedTableOptions.SetBlockRestartInterval(10)
	blockBasedTableOptions.SetBlockCacheCompressed(gorocksdb.NewLRUCache(64 * 1024))
	blockBasedTableOptions.SetCacheIndexAndFilterBlocks(true)
	blockBasedTableOptions.SetIndexType(gorocksdb.KHashSearchIndexType)

	options.SetBlockBasedTableFactory(blockBasedTableOptions)
	options.SetPrefixExtractor(gorocksdb.NewFixedPrefixTransform(3))
	options.SetAllowConcurrentMemtableWrites(false)
	options.SetCreateIfMissing(true)

	db, err := gorocksdb.OpenDb(options, dbPath)
	if err != nil {
		return nil, err
	}
	return &RocksDB{db: db}, nil
}

//Get get
func (db *RocksDB) Get(key []byte) ([]byte, error) {
	res, err := db.db.Get(gorocksdb.NewDefaultReadOptions(), key)
	if err != nil {
		rocksLog.Error("Get", "error", err)
		return nil, err
	}
	if !res.Exists() {
		return nil, ErrNotFoundInDb
	}
	defer res.Free()
	return cloneByte(res.Data()), nil
}

//Set set
func (db *RocksDB) Set(key []byte, value []byte) error {
	err := db.db.Put(gorocksdb.NewDefaultWriteOptions(), key, value)
	if err != nil {
		rocksLog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync 同步set
func (db *RocksDB) SetSync(key []byte, value []byte) error {
	opts := gorocksdb.NewDefaultWriteOptions()
	opts.SetSync(true)
	err := db.db.Put(opts, key, value)
	if err != nil {
		rocksLog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete 删除
func (db *RocksDB) Delete(key []byte) error {
	err := db.db.Delete(gorocksdb.NewDefaultWriteOptions(), key)
	if err != nil {
		rocksLog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync 同步删除
func (db *RocksDB) DeleteSync(key []byte) error {
	opts := gorocksdb.NewDefaultWriteOptions()
	opts.SetSync(true)
	err := db.db.Delete(opts, key)
	if err != nil {
		rocksLog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *RocksDB) DB() *gorocksdb.DB {
	return db.db
}

//Close 关闭
func (db *RocksDB) Close() {
	db.db.Close()
}

//Print 打印
func (db *RocksDB) Print() {}

//Stats ...
func (db *RocksDB) Stats() map[string]string { return nil }

//Iterator 迭代器
func (db *RocksDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil && len(start) != 0 {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	opts := gorocksdb.NewDefaultReadOptions()
	if len(end) != 0 {
		fmt.Println("Iterator end:", string(end))
		opts.SetIterateUpperBound(end)
	}

	it := db.db.NewIterator(opts)

	return &rocksIt{it, itBase{start, end, reverse}}
}

// CompactRange ...
func (db *RocksDB) CompactRange(start, limit []byte) error {
	db.db.CompactRange(gorocksdb.Range{
		Start: start,
		Limit: limit,
	})
	return nil
}

type rocksIt struct {
	*gorocksdb.Iterator
	itBase
}

//Rewind ...
func (it *rocksIt) Rewind() bool {
	if it.reverse {
		it.Iterator.SeekToLast()
	} else if len(it.start) != 0 {
		it.Iterator.Seek(it.start)
	} else {
		it.Iterator.SeekToFirst()
	}
	return it.Valid()
}

// Seek seek
func (it *rocksIt) Seek(key []byte) bool {
	it.Iterator.Seek(key)
	return it.Valid()
}

//Next next
func (it *rocksIt) Next() bool {
	if it.reverse {
		it.Iterator.Prev()
	} else {
		it.Iterator.Next()
	}
	return it.Valid()
}

// Valid valid
func (it *rocksIt) Valid() bool {
	return it.Iterator.Valid() && it.checkKey(it.Key())
}

// Key key
func (it *rocksIt) Key() []byte {
	return it.Iterator.Key().Data()
}

// Value value
func (it *rocksIt) Value() []byte {
	return it.Iterator.Value().Data()
}

// ValueCopy copy
func (it *rocksIt) ValueCopy() []byte {
	v := it.Value()
	return cloneByte(v)
}

// Error error
func (it *rocksIt) Error() error {
	return it.Iterator.Err()
}

//Close 关闭
func (it *rocksIt) Close() {
	it.Iterator.Close()
}

type rocksBatch struct {
	db    *RocksDB
	batch *gorocksdb.WriteBatch
	wop   *gorocksdb.WriteOptions
	size  int
	len   int
}

//NewBatch new
func (db *RocksDB) NewBatch(sync bool) Batch {
	opts := gorocksdb.NewDefaultWriteOptions()
	opts.SetSync(sync)
	batch := &rocksBatch{
		db:    db,
		batch: gorocksdb.NewWriteBatch(),
		wop:   opts,
		size:  0,
		len:   0,
	}
	return batch
}

// Set batch set
func (rb *rocksBatch) Set(key, value []byte) {
	rb.batch.Put(key, value)
	rb.size += len(key)
	rb.size += len(value)
	rb.len += len(value)
}

// Delete batch delete
func (rb *rocksBatch) Delete(key []byte) {
	rb.batch.Delete(key)
	rb.size += len(key)
	rb.len++
}

// Write batch write
func (rb *rocksBatch) Write() error {
	err := rb.db.db.Write(rb.wop, rb.batch)
	if err != nil {
		rocksLog.Error("Write", "error", err)
		return err
	}
	return nil
}

// ValueSize size of batch
func (rb *rocksBatch) ValueSize() int {
	return rb.size
}

// ValueLen size of batch value
func (rb *rocksBatch) ValueLen() int {
	return rb.len
}

// Reset resets the batch
func (rb *rocksBatch) Reset() {
	rb.batch.Clear()
	rb.len = 0
	rb.size = 0
}

// UpdateWriteSync ...
func (rb *rocksBatch) UpdateWriteSync(sync bool) {
	rb.wop.SetSync(sync)
}

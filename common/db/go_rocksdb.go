package db

import (
	"bytes"
	"os"
	"path"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var rocksLog = log15.New("module", "db.rocksdb")

// PebbleDB db
type RocksDB struct {
	BaseDB
	db *cgoRocksdb
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

	db, err := cgoOpenRocksdb(dbPath)
	if err != nil {
		return nil, err
	}
	return &RocksDB{db: db}, nil
}

//Get get
func (db *RocksDB) Get(key []byte) ([]byte, error) {
	return db.db.get(key)
}

//Set set
func (db *RocksDB) Set(key []byte, value []byte) error {
	err := db.db.set(key, value, false)
	if err != nil {
		rocksLog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync 同步set
func (db *RocksDB) SetSync(key []byte, value []byte) error {
	err := db.db.set(key, value, true)
	if err != nil {
		rocksLog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete 删除
func (db *RocksDB) Delete(key []byte) error {
	err := db.db.delete(key, false)
	if err != nil {
		rocksLog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync 同步删除
func (db *RocksDB) DeleteSync(key []byte) error {
	err := db.db.delete(key, true)
	if err != nil {
		rocksLog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *RocksDB) DB() *cgoRocksdb {
	return db.db
}

//Close 关闭
func (db *RocksDB) Close() {
	db.db.close()
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

	it := db.db.newIter(end)

	return &rocksIt{it, itBase{start, end, reverse}}
}

// CompactRange ...
func (db *RocksDB) CompactRange(start, limit []byte) error {
	db.db.compactRange(start, limit)
	return nil
}

type rocksIt struct {
	it *cgoRocksIter
	itBase
}

//Rewind ...
func (it *rocksIt) Rewind() bool {
	if it.reverse {
		it.it.seekToLast()
	} else if len(it.start) != 0 {
		it.it.seek(it.start)
	} else {
		it.it.seekToFirst()
	}
	return it.Valid()
}

// Seek seek
func (it *rocksIt) Seek(key []byte) bool {
	it.it.seek(key)
	return it.Valid()
}

//Next next
func (it *rocksIt) Next() bool {
	if it.reverse {
		it.it.prev()
	} else {
		it.it.next()
	}
	return it.Valid()
}

// Valid valid
func (it *rocksIt) Valid() bool {
	return it.it.valid() && it.checkKey(it.Key())
}

// Key key
func (it *rocksIt) Key() []byte {
	return it.it.key()
}

// Value value
func (it *rocksIt) Value() []byte {
	return it.it.value()
}

// ValueCopy copy
func (it *rocksIt) ValueCopy() []byte {
	v := it.Value()
	return cloneByte(v)
}

// Error error
func (it *rocksIt) Error() error {
	return it.it.error()
}

//Close 关闭
func (it *rocksIt) Close() {
	it.it.close()
}

type rocksBatch struct {
	db    *RocksDB
	batch *cgoRocksWriteBatch
	sync  bool
	size  int
	len   int
}

//NewBatch new
func (db *RocksDB) NewBatch(sync bool) Batch {
	batch := &rocksBatch{
		db:    db,
		batch: db.db.newBatch(),
		sync:  sync,
	}
	return batch
}

// Set batch set
func (rb *rocksBatch) Set(key, value []byte) {
	rb.batch.put(key, value)
	rb.size += len(key)
	rb.size += len(value)
	rb.len += len(value)
}

// Delete batch delete
func (rb *rocksBatch) Delete(key []byte) {
	rb.batch.delete(key)
	rb.size += len(key)
	rb.len++
}

// Write batch write
func (rb *rocksBatch) Write() error {
	err := rb.batch.write(rb.db.db, rb.sync)
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
	rb.batch.clear()
	rb.len = 0
	rb.size = 0
}

// UpdateWriteSync ...
func (rb *rocksBatch) UpdateWriteSync(sync bool) {
	rb.sync = sync
}

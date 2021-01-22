package db

// #cgo CFLAGS: -I/home/yann/gomodule/rocksdb/include
// #cgo LDFLAGS: -L/home/yann/gomodule/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd -ldl
// #include <stdlib.h>
// #include "rocksdb/c.h"
import "C"
import (
	"reflect"
	"unsafe"

	"github.com/syndtr/goleveldb/leveldb/errors"
)

type cgoRocksdb struct {
	c *C.rocksdb_t
}

type cgoRocksIter struct {
	c *C.rocksdb_iterator_t
}

type cgoRocksWriteBatch struct {
	c *C.rocksdb_writebatch_t
}

func cgoOpenRocksdb(dbPath string) (*cgoRocksdb, error) {
	var (
		cErr  *C.char
		cName = C.CString(dbPath)
	)
	defer C.free(unsafe.Pointer(cName))

	opts := C.rocksdb_options_create()
	C.rocksdb_options_set_create_if_missing(opts, C.uchar(1))
	C.rocksdb_options_enable_statistics(opts)
	// rocksdb_options_set_max_write_buffer_number sets the maximum number of write buffers
	// that are built up in memory.
	// The default is 2, so that when 1 write buffer is being flushed to
	// storage, new writes can continue to the other write buffer.
	C.rocksdb_options_set_max_write_buffer_number(opts, C.int(3))
	// rocksdb_options_set_max_background_compactions sets the maximum number of
	// concurrent background jobs, submitted to the default LOW priority thread pool
	// Default: 1
	C.rocksdb_options_set_max_background_compactions(opts, C.int(5))
	// rocksdb_options_set_hash_skip_list_rep sets a hash skip list as MemTableRep.
	// It contains a fixed array of buckets, each
	// pointing to a skipList (null if the bucket is empty).
	// bucketCount:             number of fixed array buckets
	// skipListHeight:          the max height of the skipList
	// skipListBranchingFactor: probabilistic size ratio between adjacent
	//                          link lists in the skipList
	C.rocksdb_options_set_hash_skip_list_rep(opts, C.size_t(2e6), C.int32_t(4), C.int32_t(4))

	blockBasedTableOptions := C.rocksdb_block_based_options_create()
	C.rocksdb_block_based_options_set_block_cache(blockBasedTableOptions, C.rocksdb_cache_create_lru(C.size_t(1<<16)))
	C.rocksdb_block_based_options_set_block_cache_compressed(blockBasedTableOptions, C.rocksdb_cache_create_lru(C.size_t(1<<16)))
	C.rocksdb_block_based_options_set_filter_policy(blockBasedTableOptions, C.rocksdb_filterpolicy_create_bloom(C.int(10)))
	C.rocksdb_block_based_options_set_cache_index_and_filter_blocks(blockBasedTableOptions, C.uchar(1))
	C.rocksdb_options_set_block_based_table_factory(opts, blockBasedTableOptions)

	C.rocksdb_options_set_allow_concurrent_memtable_write(opts, C.uchar(0))

	db := C.rocksdb_open(opts, cName, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}
	return &cgoRocksdb{c: db}, nil
}

func (db *cgoRocksdb) get(key []byte) ([]byte, error) {
	var (
		cErr    *C.char
		cValLen C.size_t
		cKey    = byteToChar(key)
	)
	cValue := C.rocksdb_get(db.c, C.rocksdb_readoptions_create(), cKey, C.size_t(len(key)), &cValLen, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}
	if int(cValLen) == 0 {
		return nil, ErrNotFoundInDb
	}
	defer C.rocksdb_free(unsafe.Pointer(cValue))
	return cloneByte(charToByte(cValue, cValLen)), nil
}

func (db *cgoRocksdb) set(key, value []byte, sync bool) error {
	var (
		cErr   *C.char
		cKey   = byteToChar(key)
		cValue = byteToChar(value)
	)
	opt := C.rocksdb_writeoptions_create()
	if sync {
		C.rocksdb_writeoptions_set_sync(opt, C.uchar(1))
	}
	C.rocksdb_put(db.c, opt, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)), &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

func (db *cgoRocksdb) delete(key []byte, sync bool) error {
	var (
		cErr *C.char
		cKey = byteToChar(key)
	)
	opt := C.rocksdb_writeoptions_create()
	if sync {
		C.rocksdb_writeoptions_set_sync(opt, C.uchar(1))
	}
	C.rocksdb_delete(db.c, opt, cKey, C.size_t(len(key)), &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

func (db *cgoRocksdb) compactRange(start, limit []byte) {
	cStart := byteToChar(start)
	cLimit := byteToChar(limit)
	C.rocksdb_compact_range(db.c, cStart, C.size_t(len(start)), cLimit, C.size_t(len(limit)))
}

func (db *cgoRocksdb) close() {
	C.rocksdb_close(db.c)
}

func (db *cgoRocksdb) newIter(upperBound []byte) *cgoRocksIter {
	opt := C.rocksdb_readoptions_create()
	if upperBound != nil {
		cKey := byteToChar(upperBound)
		cKeyLen := C.size_t(len(upperBound))
		C.rocksdb_readoptions_set_iterate_upper_bound(opt, cKey, cKeyLen)
	}
	it := C.rocksdb_create_iterator(db.c, opt)
	return &cgoRocksIter{c: it}
}

func (db *cgoRocksdb) newBatch() *cgoRocksWriteBatch {
	return &cgoRocksWriteBatch{c: C.rocksdb_writebatch_create()}
}

func (it *cgoRocksIter) seek(key []byte) {
	cKey := byteToChar(key)
	C.rocksdb_iter_seek(it.c, cKey, C.size_t(len(key)))
}

func (it *cgoRocksIter) seekForPrev(key []byte) {
	cKey := byteToChar(key)
	C.rocksdb_iter_seek_for_prev(it.c, cKey, C.size_t(len(key)))
}

func (it *cgoRocksIter) seekToLast() {
	C.rocksdb_iter_seek_to_last(it.c)
}

func (it *cgoRocksIter) seekToFirst() {
	C.rocksdb_iter_seek_to_first(it.c)
}

func (it *cgoRocksIter) next() {
	C.rocksdb_iter_next(it.c)
}

func (it *cgoRocksIter) prev() {
	C.rocksdb_iter_prev(it.c)
}

func (it *cgoRocksIter) valid() bool {
	return C.rocksdb_iter_valid(it.c) != 0
}

func (it *cgoRocksIter) key() []byte {
	var cLen C.size_t
	cKey := C.rocksdb_iter_key(it.c, &cLen)
	if cKey == nil {
		return nil
	}
	return charToByte(cKey, cLen)
}

func (it *cgoRocksIter) value() []byte {
	var cLen C.size_t
	cVal := C.rocksdb_iter_value(it.c, &cLen)
	if cVal == nil {
		return nil
	}
	return charToByte(cVal, cLen)
}

func (it *cgoRocksIter) error() error {
	var cErr *C.char
	C.rocksdb_iter_get_error(it.c, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

func (it *cgoRocksIter) close() {
	C.rocksdb_iter_destroy(it.c)
}

func (wb *cgoRocksWriteBatch) put(key, value []byte) {
	cKey := byteToChar(key)
	cValue := byteToChar(value)
	C.rocksdb_writebatch_put(wb.c, cKey, C.size_t(len(key)), cValue, C.size_t(len(value)))
}

func (wb *cgoRocksWriteBatch) delete(key []byte) {
	cKey := byteToChar(key)
	C.rocksdb_writebatch_delete(wb.c, cKey, C.size_t(len(key)))
}

func (wb *cgoRocksWriteBatch) write(db *cgoRocksdb, sync bool) error {
	var cErr *C.char
	opt := C.rocksdb_writeoptions_create()
	if sync {
		C.rocksdb_writeoptions_set_sync(opt, C.uchar(1))
	}
	C.rocksdb_write(db.c, opt, wb.c, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

func (wb *cgoRocksWriteBatch) clear() {
	C.rocksdb_writebatch_clear(wb.c)
}

// byteToChar converts a byte slice to a *C.char.
func byteToChar(b []byte) *C.char {
	var c *C.char
	if len(b) > 0 {
		c = (*C.char)(unsafe.Pointer(&b[0]))
	}
	return c
}

// charToByte converts a *C.char to a byte slice.
func charToByte(data *C.char, len C.size_t) []byte {
	var value []byte
	sH := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	sH.Cap, sH.Len, sH.Data = int(len), int(len), uintptr(unsafe.Pointer(data))
	return value
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/tecbot/gorocksdb"
)

func main() {
	dir, err := ioutil.TempDir("", "rocksdb")
	if err != nil {
		panic(err)
	}
	fmt.Println("dir:", dir)
	defer os.RemoveAll(dir)
	rocksdb := openDB(dir)
	defer rocksdb.Close()

	var kvs [][]byte
	kvs = append(kvs, []byte("aaa"))
	kvs = append(kvs, []byte("bbb"))
	kvs = append(kvs, []byte("ccc"))
	kvs = append(kvs, []byte("ddd"))
	kvs = append(kvs, []byte("eee"))
	kvs = append(kvs, []byte("fff"))
	for i, k := range kvs {
		err = rocksdb.Put(gorocksdb.NewDefaultWriteOptions(), k, k)
		if err != nil {
			fmt.Println(i)
			panic(err)
		}
	}

	rop := gorocksdb.NewDefaultReadOptions()
	//rop.SetPrefixSameAsStart(false)
	//rop.SetFillCache(false)
	//rop.SetIterateUpperBound(kvs[4])
	it := rocksdb.NewIterator(rop)
	defer it.Close()

	it.Seek([]byte("bbb"))
	fmt.Println(it.Valid(), string(it.Key().Data()), string(it.Value().Data()))
	it.Seek([]byte("ddd"))
	fmt.Println(it.Valid(), string(it.Key().Data()), string(it.Value().Data()))
	it.Seek([]byte("fff"))
	fmt.Println(it.Valid(), string(it.Key().Data()), string(it.Value().Data()))
	//it.SeekToFirst()
	//fmt.Println(it.Valid(), string(it.Key().Data()), string(it.Value().Data()))
	it.SeekForPrev(kvs[4])
	fmt.Println(it.Valid())
	//k, v := it.Key(), it.Value()
	//fmt.Println("first:", string(k.Data()), v.Data(), it.Valid())
	//k.Free()
	//v.Free()
	//
	//it.Next()
	//fmt.Println(it.Valid(), it.Err())
	//fmt.Println("next:", string(it.Key().Data()))
	//it.Seek(kvs[3])
	//k, v = it.Key(), it.Value()
	//fmt.Println("last:", string(k.Data()), v.Data(), it.Valid())
	//k.Free()
	//v.Free()
	//
	//it.SeekToLast()
	//fmt.Println("last:", string(k.Data()), v.Data(), it.Valid())
	//
	//it.SeekToFirst()
	//fmt.Println("first:", string(k.Data()), v.Data(), it.Valid())
	//
	//it.Prev()
	//
	//fmt.Println(it.Valid(), it.Err())
	//k = it.Key()
	//fmt.Println("prev:", string(k.Data()))
	//k.Free()

}

func openDB(dir string) *gorocksdb.DB {
	dbPath := path.Join(dir, "rocks.db")
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
		panic(err)
	}
	return db
}

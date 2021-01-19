// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/stretchr/testify/require"
)

// rocksdb迭代器测试
func TestRocksDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)
	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()
	testDBIterator(t, rocksdb)
}

func TestRocksDBIteratorAll(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)
	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()
	testDBIteratorAllKey(t, rocksdb)
}

func TestRocksDBIteratorReserverExample(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)
	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()
	testDBIteratorReserverExample(t, rocksdb)
}

func TestRocksDBIteratorDel(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)

	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()

	testDBIteratorDel(t, rocksdb)
}

func TestRocksDBBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)

	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()
	testBatch(t, rocksdb)
}

// rocksdb边界测试
func TestRocksDBBoundary(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)

	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()

	testDBBoundary(t, rocksdb)
}

func BenchmarkRocksDBBatchWrites(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewRocksDB("rocksdb", dir, 100)
	require.Nil(b, err)
	batch := db.NewBatch(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := string(common.GetRandBytes(20, 64))
		value := fmt.Sprintf("v%d", i)
		b.StartTimer()
		batch.Set([]byte(key), []byte(value))
		if i > 0 && i%10000 == 0 {
			err := batch.Write()
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	require.Nil(b, err)
	b.StopTimer()
}

func BenchmarkRocksDBBatchWrites1k(b *testing.B) {
	benchmarkBatchWritesRocksDB(b, 1)
}

func BenchmarkRocksDBBatchWrites16k(b *testing.B) {
	benchmarkBatchWritesRocksDB(b, 16)
}

func BenchmarkRocksDBBatchWrites256k(b *testing.B) {
	benchmarkBatchWritesRocksDB(b, 256)
}

func BenchmarkRocksDBBatchWrites1M(b *testing.B) {
	benchmarkBatchWritesRocksDB(b, 1024)
}

//func BenchmarkRocksDBBatchWrites4M(b *testing.B) {
//	benchmarkBatchWritesRocksDB(b, 1024*4)
//}

func benchmarkBatchWritesRocksDB(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewRocksDB("rocksdb", dir, 100)
	require.Nil(b, err)
	batch := db.NewBatch(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		b.StartTimer()
		batch.Set(key, value)
		if i > 0 && i%100 == 0 {
			err := batch.Write()
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	require.Nil(b, err)
	b.StopTimer()
}

func BenchmarkRocksDBRandomReads1K(b *testing.B) {
	benchmarkRocksDBRandomReads(b, 1)
}

func BenchmarkRocksDBRandomReads16K(b *testing.B) {
	benchmarkRocksDBRandomReads(b, 16)
}

func BenchmarkRocksDBRandomReads256K(b *testing.B) {
	benchmarkRocksDBRandomReads(b, 256)
}

func BenchmarkRocksDBRandomReads1M(b *testing.B) {
	benchmarkRocksDBRandomReads(b, 1024)
}

func benchmarkRocksDBRandomReads(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewRocksDB("rocksdb", dir, 100)
	require.Nil(b, err)
	batch := db.NewBatch(true)
	var keys [][]byte
	for i := 0; i < 32*1024/size; i++ {
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		batch.Set(key, value)
		keys = append(keys, key)
		if batch.ValueSize() > 1<<20 {
			err := batch.Write()
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	if batch.ValueSize() > 0 {
		err = batch.Write()
		require.Nil(b, err)
	}

	//开始rand 读取
	db.Close()
	db, err = NewRocksDB("rocksdb", dir, 1)
	require.Nil(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := RandInt() % len(keys)
		key := keys[index]
		_, err := db.Get(key)
		require.Nil(b, err)
	}
	b.StopTimer()
}

func BenchmarkRocksDBRandomReadsWrites(b *testing.B) {
	numItems := int64(1000000)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	dir := fmt.Sprintf("test_%x", RandStr(12))
	defer os.RemoveAll(dir)
	db, err := NewRocksDB("rocksdb", dir, 1000)
	if err != nil {
		b.Fatal(err.Error())
		return
	}
	for i := 0; i < b.N; i++ {
		// Write something
		{
			idx := (int64(RandInt()) % numItems)
			internal[idx]++
			val := internal[idx]
			idxBytes := int642Bytes(idx)
			valBytes := int642Bytes(val)
			//fmt.Printf("Set %X -> %X\n", idxBytes, valBytes)
			db.Set(
				idxBytes,
				valBytes,
			)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Read something
		{
			idx := (int64(RandInt()) % numItems)
			val := internal[idx]
			idxBytes := int642Bytes(idx)
			valBytes, _ := db.Get(idxBytes)
			//fmt.Printf("Get %X -> %X\n", idxBytes, valBytes)
			if val == 0 {
				if !bytes.Equal(valBytes, nil) {
					b.Errorf("Expected %v for %v, got %X",
						nil, idx, valBytes)
					break
				}
			} else {
				if len(valBytes) != 8 {
					b.Errorf("Expected length 8 for %v, got %X",
						idx, valBytes)
					break
				}
				valGot := bytes2Int64(valBytes)
				if val != valGot {
					b.Errorf("Expected %v for %v, got %v",
						val, idx, valGot)
					break
				}
			}
		}
	}
	b.StopTimer()
	db.Close()
}

// rocksdb返回值测试
func TestRocksDBResult(t *testing.T) {
	dir, err := ioutil.TempDir("", "rocksdb")
	require.NoError(t, err)
	t.Log(dir)

	rocksdb, err := NewRocksDB("rocksdb", dir, 128)
	require.NoError(t, err)
	defer rocksdb.Close()

	testDBIteratorResult(t, rocksdb)
}

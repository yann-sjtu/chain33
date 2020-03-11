package store

import (
	"io/ioutil"

	dbm "github.com/33cn/chain33/common/db"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
)

func DataStoreOption() opts.Option {
	dir, err := ioutil.TempDir("", DBDir)
	if err != nil {
		return nil
	}
	return opts.Datastore(&Persistent{dbm.NewDB(DBName, DBType, dir, DBCacheSize)})
}

func ValidatorOption() opts.Option {
	return opts.NamespacedValidator(DhtStoreNamespace, &Validator{})
}

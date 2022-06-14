package tool

import (
	"strings"
	"unsafe"

	"github.com/taosdata/driver-go/v2/wrapper"
	"github.com/taosdata/taosadapter/db/async"
	"github.com/taosdata/taosadapter/thread"
	"github.com/taosdata/taosadapter/tools/pool"
)

func CreateDBWithConnection(connection unsafe.Pointer, db string) error {
	b := pool.BytesPoolGet()
	defer pool.BytesPoolPut(b)
	b.WriteString("create database if not exists ")
	b.WriteString(db)
	b.WriteString(" precision 'ns' schemaless 1")
	err := async.GlobalAsync.TaosExecWithoutResult(connection, b.String())
	if err != nil {
		return err
	}
	return nil
}

func SelectDB(taosConnect unsafe.Pointer, db string) error {
	err := async.GlobalAsync.TaosExecWithoutResult(taosConnect, "use "+db)
	if err != nil {
		if strings.Contains(err.Error(), "Database not exist") {
			err := CreateDBWithConnection(taosConnect, db)
			if err != nil {
				return err
			}
			thread.Lock()
			wrapper.TaosSelectDB(taosConnect, db)
			thread.Unlock()
		} else {
			return err
		}
	}
	return nil
}

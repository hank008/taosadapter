package capi_test

import (
	"testing"
	"unsafe"

	"github.com/sirupsen/logrus"
	"github.com/taosdata/driver-go/v3/errors"
	"github.com/taosdata/driver-go/v3/wrapper"
	"github.com/taosdata/taosadapter/v3/schemaless/capi"
	"github.com/taosdata/taosadapter/v3/schemaless/proto"
)

// @author: xftan
// @date: 2021/12/14 15:11
// @description: test insert influxdb
func TestInsertInfluxdb(t *testing.T) {
	conn, err := wrapper.TaosConnect("", "root", "taosdata", "", 0)
	if err != nil {
		t.Error(err)
		return
	}
	defer wrapper.TaosClose(conn)
	defer func() {
		r := wrapper.TaosQuery(conn, "drop database if exists test_capi")
		code := wrapper.TaosError(r)
		if code != 0 {
			errStr := wrapper.TaosErrorStr(r)
			t.Error(errors.NewError(code, errStr))
		}
		wrapper.TaosFreeResult(r)
	}()
	r := wrapper.TaosQuery(conn, "create database if not exists test_capi")
	code := wrapper.TaosError(r)
	if code != 0 {
		errStr := wrapper.TaosErrorStr(r)
		t.Error(errors.NewError(code, errStr))
	}
	wrapper.TaosFreeResult(r)
	type args struct {
		taosConnect unsafe.Pointer
		data        []byte
		db          string
		precision   string
		ttl         int
	}
	tests := []struct {
		name    string
		args    args
		want    *proto.InfluxResult
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				taosConnect: conn,
				data:        []byte("measurement,host=host1 field1=2i,field2=2.0,fieldKey=\"Launch 🚀\" 1577836800000000001"),
				db:          "test_capi",
				ttl:         0,
			},
			wantErr: false,
		}, {
			name: "wrong",
			args: args{
				taosConnect: conn,
				data:        []byte("wrong,host=host1 field1=wrong 1577836800000000001"),
				db:          "test_capi",
				ttl:         100,
			},
			wantErr: true,
		}, {
			name: "wrongdb",
			args: args{
				taosConnect: conn,
				data:        []byte("wrong,host=host1 field1=wrong 1577836800000000001"),
				db:          "1'test_capi",
				ttl:         1000,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New().WithField("test", "TestInsertInfluxdb")
			err := capi.InsertInfluxdb(tt.args.taosConnect, tt.args.data, tt.args.db, tt.args.precision, tt.args.ttl, 0, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertInfluxdb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

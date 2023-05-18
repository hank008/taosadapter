package tmq

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/taosdata/driver-go/v3/common/tmq"
	"github.com/taosdata/taosadapter/v3/config"
	"github.com/taosdata/taosadapter/v3/controller"
	_ "github.com/taosdata/taosadapter/v3/controller/rest"
	"github.com/taosdata/taosadapter/v3/controller/ws/query"
	"github.com/taosdata/taosadapter/v3/controller/ws/wstool"
	"github.com/taosdata/taosadapter/v3/db"
	"github.com/taosdata/taosadapter/v3/tools/parseblock"
)

var router *gin.Engine

func TestMain(m *testing.M) {
	viper.Set("pool.maxConnect", 10000)
	viper.Set("pool.maxIdle", 10000)
	viper.Set("logLevel", "debug")
	config.Init()
	db.PrepareConnection()
	gin.SetMode(gin.ReleaseMode)
	router = gin.New()
	controllers := controller.GetControllers()
	for _, webController := range controllers {
		webController.Init(router)
	}
	m.Run()
}

func TestTMQ(t *testing.T) {
	ts1 := time.Now()
	ts2 := ts1.Add(time.Second)
	ts3 := ts2.Add(time.Second)
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists test_ws_tmq WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct0 (ts timestamp, c1 int)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct1 (ts timestamp, c1 int, c2 float)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct2 (ts timestamp, c1 int, c2 float, c3 binary(10))")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists test_tmq_ws_topic as DATABASE test_ws_tmq")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct0 values('%s',1)`, ts1.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct1 values('%s',1,2)`, ts2.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct2 values('%s',1,2,'3')`, ts3.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/tmq", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()
	const (
		AfterTMQSubscribe = iota + 1
		AfterTMQPoll
		AfterTMQFetch
		AfterTMQFetchBlock
		AfterTMQCommit
		AfterVersion
	)
	messageID := uint64(0)
	status := 0
	finish := make(chan struct{})
	var tmqFetchResp *TMQFetchResp
	pollCount := 0
	testMessageHandler := func(messageType int, message []byte) error {
		if messageType == websocket.BinaryMessage {
			t.Log(messageType, message)
		} else {
			t.Log(messageType, string(message))
		}
		switch status {
		case AfterTMQSubscribe:
			var d TMQSubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQSubscribe, d.Code, d.Message)
			}
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQPoll:
			if pollCount == 5 {
				status = AfterVersion
				action, _ := json.Marshal(&wstool.WSAction{
					Action: wstool.ClientVersion,
					Args:   nil,
				})
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				return nil
			}
			pollCount += 1
			var d TMQPollResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQPoll, d.Code, d.Message)
			}
			if d.HaveMessage {
				messageID = d.MessageID
				status = AfterTMQFetch
				b, _ := json.Marshal(&TMQFetchReq{
					ReqID:     4,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetch,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				status = AfterTMQPoll
				//fetch
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetch:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}

			if d.Completed {
				status = AfterTMQCommit
				b, _ := json.Marshal(&TMQCommitReq{
					ReqID:     3,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQCommit,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				tmqFetchResp = &d
				status = AfterTMQFetchBlock
				b, _ := json.Marshal(&TMQFetchBlockReq{
					ReqID:     0,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetchBlock,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetchBlock:
			_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
			switch tmqFetchResp.TableName {
			case "ct0":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts1.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
			case "ct1":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts2.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
			case "ct2":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts3.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
				assert.Equal(t, "3", value[0][3])
			}
			_ = value
			status = AfterTMQFetch
			b, _ := json.Marshal(&TMQFetchReq{
				ReqID:     4,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQFetch,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQCommit:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQCommit, d.Code, d.Message)
			}
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterVersion:
			var d wstool.WSVersionResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}
			assert.NotEmpty(t, d.Version)
			t.Log("client version", d.Version)
			finish <- struct{}{}
			return nil
		}
		return nil
	}
	go func() {
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				t.Error(err)
				finish <- struct{}{}
				return
			}
			err = testMessageHandler(mt, message)
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				if mt == websocket.BinaryMessage {
					t.Error(err, message)
				} else {
					t.Error(err, string(message))
				}
				finish <- struct{}{}
				return
			}
		}
	}()
	init := &TMQSubscribeReq{
		ReqID:                0,
		User:                 "root",
		Password:             "taosdata",
		GroupID:              "test",
		Topics:               []string{"test_tmq_ws_topic"},
		AutoCommit:           "true",
		AutoCommitIntervalMS: "5000",
		SnapshotEnable:       "true",
		WithTableName:        "true",
	}

	b, _ := json.Marshal(init)
	action, _ := json.Marshal(&wstool.WSAction{
		Action: TMQSubscribe,
		Args:   b,
	})
	t.Log(string(action))
	status = AfterTMQSubscribe
	err = ws.WriteMessage(
		websocket.TextMessage,
		action,
	)
	if err != nil {
		t.Error(err)
		return
	}
	<-finish
	ws.Close()
	time.Sleep(time.Second * 3)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists test_tmq_ws_topic")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists test_ws_tmq")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMeta(t *testing.T) {
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists test_ws_tmq_meta WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists test_tmq_meta_ws_topic with meta as DATABASE test_ws_tmq_meta")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_meta", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/tmq", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()
	const (
		AfterTMQSubscribe = iota + 1
		AfterTMQPoll
		AfterFetchRawMeta
		AfterFetchJsonMeta
		AfterTMQCommit
		AfterUnsubscribe
		AfterVersion
	)
	messageID := uint64(0)
	status := 0
	finish := make(chan struct{})
	pollCount := 0
	testMessageHandler := func(messageType int, message []byte) error {
		if messageType == websocket.BinaryMessage {
			t.Log(messageType, message)
		} else {
			t.Log(messageType, string(message))
		}
		switch status {
		case AfterTMQSubscribe:
			var d TMQSubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQSubscribe, d.Code, d.Message)
			}
			w = httptest.NewRecorder()
			body = strings.NewReader("create table stb (ts timestamp," +
				"c1 bool," +
				"c2 tinyint," +
				"c3 smallint," +
				"c4 int," +
				"c5 bigint," +
				"c6 tinyint unsigned," +
				"c7 smallint unsigned," +
				"c8 int unsigned," +
				"c9 bigint unsigned," +
				"c10 float," +
				"c11 double," +
				"c12 binary(20)," +
				"c13 nchar(20)" +
				")" +
				"tags(tts timestamp," +
				"tc1 bool," +
				"tc2 tinyint," +
				"tc3 smallint," +
				"tc4 int," +
				"tc5 bigint," +
				"tc6 tinyint unsigned," +
				"tc7 smallint unsigned," +
				"tc8 int unsigned," +
				"tc9 bigint unsigned," +
				"tc10 float," +
				"tc11 double," +
				"tc12 binary(20)," +
				"tc13 nchar(20)" +
				")")
			req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_meta", body)
			req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQPoll:
			if pollCount == 5 {
				status = AfterUnsubscribe
				b, _ := json.Marshal(&TMQUnsubscribeReq{
					ReqID: 6,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQUnsubscribe,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
				return nil
			}
			pollCount += 1
			var d TMQPollResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQPoll, d.Code, d.Message)
			}
			if d.HaveMessage {
				messageID = d.MessageID
				status = AfterFetchJsonMeta
				b, _ := json.Marshal(&TMQFetchJsonMetaReq{
					ReqID:     4,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetchJsonMeta,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				status = AfterTMQPoll
				//fetch
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterFetchJsonMeta:
			var d TMQFetchJsonMetaResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}
			var meta tmq.Meta
			err = jsoniter.Unmarshal(d.Data, &meta)
			assert.NoError(t, err)
			t.Log(meta)
			status = AfterFetchRawMeta
			b, _ := json.Marshal(&TMQFetchRawMetaReq{
				ReqID:     3,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQFetchRaw,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterFetchRawMeta:
			if messageType != websocket.BinaryMessage {
				t.Fatal(string(message))
			}
			writeRaw(t, message)
			w = httptest.NewRecorder()
			body = strings.NewReader("describe stb")
			req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_meta_target", body)
			req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			var resp wstool.TDEngineRestfulResp
			err = jsoniter.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, [][]driver.Value{
				{"ts", "TIMESTAMP", float64(8), ""},
				{"c1", "BOOL", float64(1), ""},
				{"c2", "TINYINT", float64(1), ""},
				{"c3", "SMALLINT", float64(2), ""},
				{"c4", "INT", float64(4), ""},
				{"c5", "BIGINT", float64(8), ""},
				{"c6", "TINYINT UNSIGNED", float64(1), ""},
				{"c7", "SMALLINT UNSIGNED", float64(2), ""},
				{"c8", "INT UNSIGNED", float64(4), ""},
				{"c9", "BIGINT UNSIGNED", float64(8), ""},
				{"c10", "FLOAT", float64(4), ""},
				{"c11", "DOUBLE", float64(8), ""},
				{"c12", "VARCHAR", float64(20), ""},
				{"c13", "NCHAR", float64(20), ""},
				{"tts", "TIMESTAMP", float64(8), "TAG"},
				{"tc1", "BOOL", float64(1), "TAG"},
				{"tc2", "TINYINT", float64(1), "TAG"},
				{"tc3", "SMALLINT", float64(2), "TAG"},
				{"tc4", "INT", float64(4), "TAG"},
				{"tc5", "BIGINT", float64(8), "TAG"},
				{"tc6", "TINYINT UNSIGNED", float64(1), "TAG"},
				{"tc7", "SMALLINT UNSIGNED", float64(2), "TAG"},
				{"tc8", "INT UNSIGNED", float64(4), "TAG"},
				{"tc9", "BIGINT UNSIGNED", float64(8), "TAG"},
				{"tc10", "FLOAT", float64(4), "TAG"},
				{"tc11", "DOUBLE", float64(8), "TAG"},
				{"tc12", "VARCHAR", float64(20), "TAG"},
				{"tc13", "NCHAR", float64(20), "TAG"},
			}, resp.Data)
			status = AfterTMQCommit
			b, _ := json.Marshal(&TMQCommitReq{
				ReqID:     3,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQCommit,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQCommit:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQCommit, d.Code, d.Message)
			}
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterUnsubscribe:
			var d TMQUnsubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQUnsubscribe, d.Code, d.Message)
			}
			status = AfterVersion
			action, _ := json.Marshal(&wstool.WSAction{
				Action: wstool.ClientVersion,
				Args:   nil,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
		case AfterVersion:
			var d wstool.WSVersionResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}
			assert.NotEmpty(t, d.Version)
			t.Log("client version", d.Version)
			finish <- struct{}{}
			return nil
		}
		return nil
	}
	go func() {
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				t.Error(err)
				finish <- struct{}{}
				return
			}
			err = testMessageHandler(mt, message)
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				if mt == websocket.BinaryMessage {
					t.Error(err, message)
				} else {
					t.Error(err, string(message))
				}
				finish <- struct{}{}
				return
			}
		}
	}()
	init := &TMQSubscribeReq{
		ReqID:                0,
		User:                 "root",
		Password:             "taosdata",
		GroupID:              "test",
		Topics:               []string{"test_tmq_meta_ws_topic"},
		AutoCommit:           "true",
		AutoCommitIntervalMS: "5000",
		SnapshotEnable:       "true",
		WithTableName:        "true",
	}

	b, _ := json.Marshal(init)
	action, _ := json.Marshal(&wstool.WSAction{
		Action: TMQSubscribe,
		Args:   b,
	})
	t.Log(string(action))
	status = AfterTMQSubscribe
	err = ws.WriteMessage(
		websocket.TextMessage,
		action,
	)
	if err != nil {
		t.Error(err)
		return
	}

	<-finish
	ws.Close()
	w = httptest.NewRecorder()
	body = strings.NewReader("describe stb")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_meta_target", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var resp wstool.TDEngineRestfulResp
	err = jsoniter.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, [][]driver.Value{
		{"ts", "TIMESTAMP", float64(8), ""},
		{"c1", "BOOL", float64(1), ""},
		{"c2", "TINYINT", float64(1), ""},
		{"c3", "SMALLINT", float64(2), ""},
		{"c4", "INT", float64(4), ""},
		{"c5", "BIGINT", float64(8), ""},
		{"c6", "TINYINT UNSIGNED", float64(1), ""},
		{"c7", "SMALLINT UNSIGNED", float64(2), ""},
		{"c8", "INT UNSIGNED", float64(4), ""},
		{"c9", "BIGINT UNSIGNED", float64(8), ""},
		{"c10", "FLOAT", float64(4), ""},
		{"c11", "DOUBLE", float64(8), ""},
		{"c12", "VARCHAR", float64(20), ""},
		{"c13", "NCHAR", float64(20), ""},
		{"tts", "TIMESTAMP", float64(8), "TAG"},
		{"tc1", "BOOL", float64(1), "TAG"},
		{"tc2", "TINYINT", float64(1), "TAG"},
		{"tc3", "SMALLINT", float64(2), "TAG"},
		{"tc4", "INT", float64(4), "TAG"},
		{"tc5", "BIGINT", float64(8), "TAG"},
		{"tc6", "TINYINT UNSIGNED", float64(1), "TAG"},
		{"tc7", "SMALLINT UNSIGNED", float64(2), "TAG"},
		{"tc8", "INT UNSIGNED", float64(4), "TAG"},
		{"tc9", "BIGINT UNSIGNED", float64(8), "TAG"},
		{"tc10", "FLOAT", float64(4), "TAG"},
		{"tc11", "DOUBLE", float64(8), "TAG"},
		{"tc12", "VARCHAR", float64(20), "TAG"},
		{"tc13", "NCHAR", float64(20), "TAG"},
	}, resp.Data)
	time.Sleep(time.Second * 3)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists test_tmq_meta_ws_topic")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists test_ws_tmq_meta_target")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists test_ws_tmq_meta")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func writeRaw(t *testing.T, rawData []byte) {
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists test_ws_tmq_meta_target WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/ws", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()
	const (
		AfterConnect  = 1
		AfterWriteRaw = 2
	)

	status := 0
	//total := 0
	finish := make(chan struct{})
	//var jsonResult [][]interface{}
	testMessageHandler := func(messageType int, message []byte) error {
		//json
		switch status {
		case AfterConnect:
			var d query.WSConnectResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", query.WSConnect, d.Code, d.Message)
			}
			//query
			status = AfterWriteRaw
			err = ws.WriteMessage(websocket.BinaryMessage, rawData[8:])
			if err != nil {
				return err
			}
		case AfterWriteRaw:
			var d query.WSWriteMetaResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", query.WSQuery, d.Code, d.Message)
			}
			finish <- struct{}{}
		}
		return nil
	}
	go func() {
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				t.Error(err)
				finish <- struct{}{}
				return
			}
			err = testMessageHandler(mt, message)
			if err != nil {
				if mt == websocket.BinaryMessage {
					t.Error(err, message)
				} else {
					t.Error(err, string(message))
				}
				finish <- struct{}{}
				return
			}
		}
	}()

	connect := &query.WSConnectReq{
		ReqID:    0,
		User:     "root",
		Password: "taosdata",
		DB:       "test_ws_tmq_meta_target",
	}

	b, _ := json.Marshal(connect)
	action, _ := json.Marshal(&wstool.WSAction{
		Action: query.WSConnect,
		Args:   b,
	})
	status = AfterConnect
	err = ws.WriteMessage(
		websocket.TextMessage,
		action,
	)
	if err != nil {
		t.Error(err)
		return
	}
	<-finish
}

func TestTMQAutoCommit(t *testing.T) {
	ts1 := time.Now()
	ts2 := ts1.Add(time.Second)
	ts3 := ts2.Add(time.Second)
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists test_ws_tmq_auto_commit WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct0 (ts timestamp, c1 int)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct1 (ts timestamp, c1 int, c2 float)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct2 (ts timestamp, c1 int, c2 float, c3 binary(10))")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists test_tmq_ws_auto_commit_topic as DATABASE test_ws_tmq_auto_commit")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct0 values('%s',1)`, ts1.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct1 values('%s',1,2)`, ts2.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct2 values('%s',1,2,'3')`, ts3.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_auto_commit", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/tmq", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()
	const (
		AfterTMQSubscribe = iota + 1
		AfterTMQPoll
		AfterTMQFetch
		AfterTMQFetchBlock
		AfterVersion
	)
	messageID := uint64(0)
	status := 0
	finish := make(chan struct{})
	var tmqFetchResp *TMQFetchResp
	pollCount := 0
	expectError := false
	testMessageHandler := func(messageType int, message []byte) error {
		if messageType == websocket.BinaryMessage {
			t.Log(messageType, message)
		} else {
			t.Log(messageType, string(message))
		}
		switch status {
		case AfterTMQSubscribe:
			var d TMQSubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQSubscribe, d.Code, d.Message)
			}
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQPoll:
			if pollCount == 5 {
				status = AfterVersion
				action, _ := json.Marshal(&wstool.WSAction{
					Action: wstool.ClientVersion,
					Args:   nil,
				})
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				return nil
			}
			pollCount += 1
			var d TMQPollResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQPoll, d.Code, d.Message)
			}
			if d.HaveMessage {
				messageID = d.MessageID
				status = AfterTMQFetch
				b, _ := json.Marshal(&TMQFetchReq{
					ReqID:     4,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetch,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				status = AfterTMQPoll
				//fetch
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetch:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				if expectError {
					assert.Equal(t, d.Message, "message is nil")
					status = AfterVersion
					action, _ := json.Marshal(&wstool.WSAction{
						Action: wstool.ClientVersion,
						Args:   nil,
					})
					err = ws.WriteMessage(
						websocket.TextMessage,
						action,
					)
					return nil
				}
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}

			if d.Completed {
				status = AfterTMQPoll
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				tmqFetchResp = &d
				status = AfterTMQFetchBlock
				b, _ := json.Marshal(&TMQFetchBlockReq{
					ReqID:     0,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetchBlock,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetchBlock:
			_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
			switch tmqFetchResp.TableName {
			case "ct0":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts1.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
			case "ct1":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts2.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
			case "ct2":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts3.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
				assert.Equal(t, "3", value[0][3])
			}
			_ = value
			status = AfterTMQFetch
			b, _ := json.Marshal(&TMQFetchReq{
				ReqID:     4,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQFetch,
				Args:   b,
			})
			t.Log(string(action))
			time.Sleep(3 * 500 * time.Millisecond)
			expectError = true
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterVersion:
			var d wstool.WSVersionResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}
			assert.NotEmpty(t, d.Version)
			t.Log("client version", d.Version)

			finish <- struct{}{}
			return nil
		}
		return nil
	}
	go func() {
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				t.Error(err)
				finish <- struct{}{}
				return
			}
			err = testMessageHandler(mt, message)
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				if mt == websocket.BinaryMessage {
					t.Error(err, message)
				} else {
					t.Error(err, string(message))
				}
				finish <- struct{}{}
				return
			}
		}
	}()
	init := &TMQSubscribeReq{
		ReqID:                0,
		User:                 "root",
		Password:             "taosdata",
		GroupID:              "test",
		Topics:               []string{"test_tmq_ws_auto_commit_topic"},
		AutoCommit:           "true",
		AutoCommitIntervalMS: "500",
		SnapshotEnable:       "true",
		WithTableName:        "true",
	}

	b, _ := json.Marshal(init)
	action, _ := json.Marshal(&wstool.WSAction{
		Action: TMQSubscribe,
		Args:   b,
	})
	t.Log(string(action))
	status = AfterTMQSubscribe
	err = ws.WriteMessage(
		websocket.TextMessage,
		action,
	)
	if err != nil {
		t.Error(err)
		return
	}
	<-finish
	assert.Equal(t, true, expectError)
	ws.Close()
	time.Sleep(time.Second * 3)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists test_tmq_ws_auto_commit_topic")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists test_ws_tmq_auto_commit")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestTMQUnsubscribeAndSubscribe(t *testing.T) {
	ts1 := time.Now()
	ts2 := ts1.Add(time.Second)
	ts3 := ts2.Add(time.Second)
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists test_ws_tmq_unsubscribe WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct0 (ts timestamp, c1 int)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct1 (ts timestamp, c1 int, c2 float)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct2 (ts timestamp, c1 int, c2 float, c3 binary(10))")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists test_tmq_ws_unsubscribe_topic as DATABASE test_ws_tmq_unsubscribe")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct0 values('%s',1)`, ts1.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists test_tmq_ws_unsubscribe2_topic as select * from ct0")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct1 values('%s',1,2)`, ts2.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader(fmt.Sprintf(`insert into ct2 values('%s',1,2,'3')`, ts3.Format(time.RFC3339Nano)))
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/test_ws_tmq_unsubscribe", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/tmq", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()
	const (
		AfterTMQSubscribe = iota + 1
		AfterTMQPoll
		AfterTMQFetch
		AfterTMQFetchBlock
		AfterVersion
		AfterUnsubscribe
		AfterTMQSubscribe2
		AfterTMQPoll2
		AfterTMQFetch2
		AfterTMQFetchBlock2
	)
	messageID := uint64(0)
	status := 0
	finish := make(chan struct{})
	var tmqFetchResp *TMQFetchResp
	pollCount := 0
	testMessageHandler := func(messageType int, message []byte) error {
		if messageType == websocket.BinaryMessage {
			t.Log(messageType, message)
		} else {
			t.Log(messageType, string(message))
		}
		switch status {
		case AfterTMQSubscribe:
			var d TMQSubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQSubscribe, d.Code, d.Message)
			}
			status = AfterTMQPoll
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQPoll:
			if pollCount == 5 {
				status = AfterUnsubscribe
				b, _ := json.Marshal(&TMQUnsubscribeReq{
					ReqID: 6,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQUnsubscribe,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
				return nil
			}
			pollCount += 1
			var d TMQPollResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQPoll, d.Code, d.Message)
			}
			if d.HaveMessage {
				messageID = d.MessageID
				status = AfterTMQFetch
				b, _ := json.Marshal(&TMQFetchReq{
					ReqID:     4,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetch,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				status = AfterTMQPoll
				//fetch
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetch:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}

			if d.Completed {
				status = AfterTMQPoll
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				tmqFetchResp = &d
				status = AfterTMQFetchBlock
				b, _ := json.Marshal(&TMQFetchBlockReq{
					ReqID:     0,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetchBlock,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetchBlock:
			_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
			switch tmqFetchResp.TableName {
			case "ct0":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts1.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
			case "ct1":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts2.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
			case "ct2":
				assert.Equal(t, 1, len(value))
				assert.Equal(t, ts3.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
				assert.Equal(t, int32(1), value[0][1])
				assert.Equal(t, float32(2), value[0][2])
				assert.Equal(t, "3", value[0][3])
			}
			_ = value
			status = AfterTMQFetch
			b, _ := json.Marshal(&TMQFetchReq{
				ReqID:     4,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQFetch,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterUnsubscribe:
			pollCount = 0
			var d TMQUnsubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQUnsubscribe, d.Code, d.Message)
			}
			status = AfterTMQSubscribe2
			b, _ := json.Marshal(&TMQSubscribeReq{
				ReqID:  0,
				Topics: []string{"test_tmq_ws_unsubscribe2_topic"},
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQSubscribe,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
		case AfterTMQSubscribe2:
			var d TMQSubscribeResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQSubscribe, d.Code, d.Message)
			}
			status = AfterTMQPoll2
			b, _ := json.Marshal(&TMQPollReq{
				ReqID:        3,
				BlockingTime: 500,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterTMQPoll2:
			if pollCount == 5 {
				status = AfterVersion
				action, _ := json.Marshal(&wstool.WSAction{
					Action: wstool.ClientVersion,
					Args:   nil,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
				return nil
			}
			pollCount += 1
			var d TMQPollResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQPoll, d.Code, d.Message)
			}
			if d.HaveMessage {
				messageID = d.MessageID
				status = AfterTMQFetch2
				b, _ := json.Marshal(&TMQFetchReq{
					ReqID:     4,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetch,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				status = AfterTMQPoll2
				//fetch
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetch2:
			var d TMQFetchResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}

			if d.Completed {
				status = AfterTMQPoll2
				b, _ := json.Marshal(&TMQPollReq{
					ReqID:        3,
					BlockingTime: 500,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQPoll,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			} else {
				tmqFetchResp = &d
				status = AfterTMQFetchBlock2
				b, _ := json.Marshal(&TMQFetchBlockReq{
					ReqID:     0,
					MessageID: messageID,
				})
				action, _ := json.Marshal(&wstool.WSAction{
					Action: TMQFetchBlock,
					Args:   b,
				})
				t.Log(string(action))
				err = ws.WriteMessage(
					websocket.TextMessage,
					action,
				)
				if err != nil {
					return err
				}
			}
		case AfterTMQFetchBlock2:
			_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
			assert.Equal(t, 1, len(value))
			assert.Equal(t, ts1.UnixNano()/1e6, value[0][0].(time.Time).UnixNano()/1e6)
			assert.Equal(t, int32(1), value[0][1])
			status = AfterTMQFetch2
			b, _ := json.Marshal(&TMQFetchReq{
				ReqID:     4,
				MessageID: messageID,
			})
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQFetch,
				Args:   b,
			})
			t.Log(string(action))
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			if err != nil {
				return err
			}
		case AfterVersion:
			var d wstool.WSVersionResp
			err = json.Unmarshal(message, &d)
			if err != nil {
				return err
			}
			if d.Code != 0 {
				return fmt.Errorf("%s %d,%s", TMQFetch, d.Code, d.Message)
			}
			assert.NotEmpty(t, d.Version)
			t.Log("client version", d.Version)
			finish <- struct{}{}
			return nil
		}
		return nil
	}
	go func() {
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				t.Error(err)
				finish <- struct{}{}
				return
			}
			err = testMessageHandler(mt, message)
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					finish <- struct{}{}
					return
				}
				if mt == websocket.BinaryMessage {
					t.Error(err, message)
				} else {
					t.Error(err, string(message))
				}
				finish <- struct{}{}
				return
			}
		}
	}()
	init := &TMQSubscribeReq{
		ReqID:                0,
		User:                 "root",
		Password:             "taosdata",
		GroupID:              "test",
		Topics:               []string{"test_tmq_ws_unsubscribe_topic"},
		AutoCommit:           "true",
		AutoCommitIntervalMS: "500",
		SnapshotEnable:       "true",
		WithTableName:        "true",
	}

	b, _ := json.Marshal(init)
	action, _ := json.Marshal(&wstool.WSAction{
		Action: TMQSubscribe,
		Args:   b,
	})
	t.Log(string(action))
	status = AfterTMQSubscribe
	err = ws.WriteMessage(
		websocket.TextMessage,
		action,
	)
	if err != nil {
		t.Error(err)
		return
	}
	<-finish
	ws.Close()
	time.Sleep(time.Second * 3)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists test_tmq_ws_unsubscribe_topic")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists test_tmq_ws_unsubscribe2_topic")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists test_ws_tmq_unsubscribe")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestTMQSeek(t *testing.T) {
	vgroups := 2
	ts1 := time.Now()
	ts2 := ts1.Add(time.Second)
	ts3 := ts2.Add(time.Second)
	insertSql := []string{
		fmt.Sprintf(`insert into ct0 values('%s',1)`, ts1.Format(time.RFC3339Nano)),
		fmt.Sprintf(`insert into ct1 values('%s',1,2)`, ts2.Format(time.RFC3339Nano)),
		fmt.Sprintf(`insert into ct2 values('%s',1,2,'3')`, ts3.Format(time.RFC3339Nano)),
	}
	insertCount := len(insertSql)
	tryPollCount := 3 * insertCount
	topic := "test_tmq_ws_seek_topic"
	dbName := "test_ws_tmq_seek"
	w := httptest.NewRecorder()
	body := strings.NewReader("create database if not exists " + dbName + " vgroups " + strconv.Itoa(vgroups) + " WAL_RETENTION_PERIOD 86400")
	req, _ := http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct0 (ts timestamp, c1 int)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/"+dbName, body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct1 (ts timestamp, c1 int, c2 float)")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/"+dbName, body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	body = strings.NewReader("create table if not exists ct2 (ts timestamp, c1 int, c2 float, c3 binary(10))")
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/"+dbName, body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	for i := 0; i < insertCount; i++ {
		w = httptest.NewRecorder()
		body = strings.NewReader(insertSql[i])
		req, _ = http.NewRequest(http.MethodPost, "/rest/sql/"+dbName, body)
		req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	w = httptest.NewRecorder()
	body = strings.NewReader("create topic if not exists " + topic + " as database " + dbName)
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql/"+dbName, body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	s := httptest.NewServer(router)
	defer s.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/rest/tmq", nil)
	if err != nil {
		t.Error(err)
		return
	}

	//sub
	{
		req := &TMQSubscribeReq{
			ReqID:         0,
			User:          "root",
			Password:      "taosdata",
			GroupID:       "test",
			Topics:        []string{topic},
			AutoCommit:    "false",
			WithTableName: "true",
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQSubscribe,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQSubscribeResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
	}
	//assignment 1
	vgID := make([]int32, vgroups)
	{
		req := TMQGetTopicAssignmentReq{
			ReqID: 1,
			Topic: topic,
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQGetTopicAssignment,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQGetTopicAssignmentResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, vgroups, len(resp.Assignment))
		for i := 0; i < vgroups; i++ {
			assert.Equal(t, int64(0), resp.Assignment[i].Offset)
			assert.Equal(t, int64(0), resp.Assignment[i].Begin)
			vgID[i] = resp.Assignment[i].VGroupID
		}
	}
	//poll 1
	{
		rowCount := 0
		for i := 0; i < tryPollCount; i++ {
			if rowCount >= insertCount {
				break
			}
			req := TMQPollReq{
				ReqID:        1,
				BlockingTime: 500,
			}
			b, _ := json.Marshal(req)
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			assert.NoError(t, err)
			t.Log("send:", string(action))
			mt, message, err := ws.ReadMessage()
			assert.NoError(t, err)
			assert.Equal(t, websocket.TextMessage, mt)
			t.Log("recv:", string(message))
			var resp TMQPollResp
			err = json.Unmarshal(message, &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)
			if resp.HaveMessage {
				for {
					req := TMQFetchReq{
						ReqID:     1,
						MessageID: resp.MessageID,
					}
					b, _ := json.Marshal(req)
					action, _ := json.Marshal(&wstool.WSAction{
						Action: TMQFetch,
						Args:   b,
					})
					err = ws.WriteMessage(
						websocket.TextMessage,
						action,
					)
					assert.NoError(t, err)
					t.Log("send:", string(action))
					mt, message, err := ws.ReadMessage()
					assert.NoError(t, err)
					assert.Equal(t, websocket.TextMessage, mt)
					t.Log("recv:", string(message))
					var tmqFetchResp TMQFetchResp
					err = json.Unmarshal(message, &tmqFetchResp)
					assert.NoError(t, err)
					assert.Equal(t, 0, tmqFetchResp.Code)
					if tmqFetchResp.Completed {
						break
					} else {
						req := TMQFetchBlockReq{
							ReqID:     1,
							MessageID: tmqFetchResp.MessageID,
						}
						b, _ := json.Marshal(req)
						action, _ := json.Marshal(&wstool.WSAction{
							Action: TMQFetchBlock,
							Args:   b,
						})
						err = ws.WriteMessage(
							websocket.TextMessage,
							action,
						)
						assert.NoError(t, err)
						t.Log("send:", string(action))
						mt, message, err := ws.ReadMessage()
						assert.NoError(t, err)
						assert.Equal(t, websocket.BinaryMessage, mt)
						t.Log("recv:", message)
						_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
						t.Log(value)
						rowCount += 1
					}
				}
				{
					req := TMQCommitReq{
						ReqID:     1,
						MessageID: resp.MessageID,
					}
					b, _ := json.Marshal(req)
					action, _ := json.Marshal(&wstool.WSAction{
						Action: TMQCommit,
						Args:   b,
					})
					err = ws.WriteMessage(
						websocket.TextMessage,
						action,
					)
					assert.NoError(t, err)
					t.Log("send:", string(action))
					mt, message, err := ws.ReadMessage()
					assert.NoError(t, err)
					assert.Equal(t, websocket.TextMessage, mt)
					t.Log("recv:", string(message))
					var resp TMQPollResp
					err = json.Unmarshal(message, &resp)
					assert.NoError(t, err)
					assert.Equal(t, 0, resp.Code)
				}
			}
		}
		assert.Equal(t, insertCount, rowCount)
	}
	//assignment after poll
	{
		req := TMQGetTopicAssignmentReq{
			ReqID: 1,
			Topic: topic,
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQGetTopicAssignment,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQGetTopicAssignmentResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, vgroups, len(resp.Assignment))
		for i := 0; i < vgroups; i++ {
			assert.Greater(t, resp.Assignment[0].Offset, int64(0))
			assert.Equal(t, int64(0), resp.Assignment[0].Begin)
		}
	}
	//seek
	for i := 0; i < vgroups; i++ {
		req := TMQOffsetSeekReq{
			ReqID:    uint64(i),
			Topic:    topic,
			VgroupID: vgID[i],
			Offset:   0,
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQSeek,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQOffsetSeekResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
	}
	//assignment after seek
	{
		req := TMQGetTopicAssignmentReq{
			ReqID: 1,
			Topic: topic,
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQGetTopicAssignment,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQGetTopicAssignmentResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, vgroups, len(resp.Assignment))
		for i := 0; i < vgroups; i++ {
			assert.Equal(t, int64(0), resp.Assignment[i].Offset)
			assert.Equal(t, int64(0), resp.Assignment[i].Begin)
		}

	}
	//poll after seek
	{
		rowCount := 0
		for i := 0; i < tryPollCount; i++ {
			if rowCount >= insertCount {
				break
			}
			req := TMQPollReq{
				ReqID:        1,
				BlockingTime: 500,
			}
			b, _ := json.Marshal(req)
			action, _ := json.Marshal(&wstool.WSAction{
				Action: TMQPoll,
				Args:   b,
			})
			err = ws.WriteMessage(
				websocket.TextMessage,
				action,
			)
			assert.NoError(t, err)
			t.Log("send:", string(action))
			mt, message, err := ws.ReadMessage()
			assert.NoError(t, err)
			assert.Equal(t, websocket.TextMessage, mt)
			t.Log("recv:", string(message))
			var resp TMQPollResp
			err = json.Unmarshal(message, &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)
			if resp.HaveMessage {
				for {
					req := TMQFetchReq{
						ReqID:     1,
						MessageID: resp.MessageID,
					}
					b, _ := json.Marshal(req)
					action, _ := json.Marshal(&wstool.WSAction{
						Action: TMQFetch,
						Args:   b,
					})
					err = ws.WriteMessage(
						websocket.TextMessage,
						action,
					)
					assert.NoError(t, err)
					t.Log("send:", string(action))
					mt, message, err := ws.ReadMessage()
					assert.NoError(t, err)
					assert.Equal(t, websocket.TextMessage, mt)
					t.Log("recv:", string(message))
					var tmqFetchResp TMQFetchResp
					err = json.Unmarshal(message, &tmqFetchResp)
					assert.NoError(t, err)
					assert.Equal(t, 0, tmqFetchResp.Code)
					if tmqFetchResp.Completed {
						break
					} else {
						req := TMQFetchBlockReq{
							ReqID:     1,
							MessageID: tmqFetchResp.MessageID,
						}
						b, _ := json.Marshal(req)
						action, _ := json.Marshal(&wstool.WSAction{
							Action: TMQFetchBlock,
							Args:   b,
						})
						err = ws.WriteMessage(
							websocket.TextMessage,
							action,
						)
						assert.NoError(t, err)
						t.Log("send:", string(action))
						mt, message, err := ws.ReadMessage()
						assert.NoError(t, err)
						assert.Equal(t, websocket.BinaryMessage, mt)
						t.Log("recv:", message)
						_, _, value := parseblock.ParseTmqBlock(message[8:], tmqFetchResp.FieldsTypes, tmqFetchResp.Rows, tmqFetchResp.Precision)
						t.Log(value)
						rowCount += 1
					}
				}
				{
					req := TMQCommitReq{
						ReqID:     1,
						MessageID: resp.MessageID,
					}
					b, _ := json.Marshal(req)
					action, _ := json.Marshal(&wstool.WSAction{
						Action: TMQCommit,
						Args:   b,
					})
					err = ws.WriteMessage(
						websocket.TextMessage,
						action,
					)
					assert.NoError(t, err)
					t.Log("send:", string(action))
					mt, message, err := ws.ReadMessage()
					assert.NoError(t, err)
					assert.Equal(t, websocket.TextMessage, mt)
					t.Log("recv:", string(message))
					var resp TMQPollResp
					err = json.Unmarshal(message, &resp)
					assert.NoError(t, err)
					assert.Equal(t, 0, resp.Code)
				}
			}
		}
		assert.Equal(t, insertCount, rowCount)
	}
	//assignment after poll2
	{
		req := TMQGetTopicAssignmentReq{
			ReqID: 1,
			Topic: topic,
		}
		b, _ := json.Marshal(req)
		action, _ := json.Marshal(&wstool.WSAction{
			Action: TMQGetTopicAssignment,
			Args:   b,
		})
		err = ws.WriteMessage(
			websocket.TextMessage,
			action,
		)
		assert.NoError(t, err)
		t.Log("send:", string(action))
		mt, message, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.TextMessage, mt)
		t.Log("recv:", string(message))
		var resp TMQGetTopicAssignmentResp
		err = json.Unmarshal(message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, vgroups, len(resp.Assignment))
		for i := 0; i < vgroups; i++ {
			assert.Equal(t, resp.Assignment[i].End, resp.Assignment[i].Offset)
			assert.Equal(t, int64(0), resp.Assignment[i].Begin)
		}
	}
	ws.Close()
	time.Sleep(time.Second * 3)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop topic if exists " + topic)
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	body = strings.NewReader("drop database if exists " + dbName)
	req, _ = http.NewRequest(http.MethodPost, "/rest/sql", body)
	req.Header.Set("Authorization", "Taosd /KfeAzX/f9na8qdtNZmtONryp201ma04bEl8LcvLUd7a8qdtNZmtONryp201ma04")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

// Go Snowflake Driver - Snowflake driver for Go's database/sql package
//
// Copyright (c) 2017 Snowflake Computing Inc. All right reserved.
//

package gosnowflake

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"sync/atomic"
)

type snowflakeConn struct {
	cfg            *Config
	rest           *snowflakeRestful
	SequeceCounter uint64
	QueryId        string
	SqlState       string
}

func (sc *snowflakeConn) isDml(v int64) bool {
	switch v {
	case StatementTypeIdDml, StatementTypeIdInsert,
		StatementTypeIdUpdate, StatementTypeIdDelete,
		StatementTypeIdMerge, StatementTypeIdMultiTableInsert:
		return true
	}
	return false
}

func (sc *snowflakeConn) exec(
  query string, noResult bool, isInternal bool, parameters []driver.Value) (*ExecResponse, error) {
	var err error
	counter := atomic.AddUint64(&sc.SequeceCounter, 1)

	req := ExecRequest{
		SqlText:    query,
		AsyncExec:  noResult,
		SequenceId: counter,
	}
	req.IsInternal = isInternal
	if len(parameters) > 0 {
		req.Bindings = make(map[string]ExecBindParameter, len(parameters))
		for i, n := 0, len(parameters); i < n; i++ {
			v1, err := valueToString(parameters[i])
			if err != nil {
				return nil, err
			}
			req.Bindings[strconv.Itoa(i+1)] = ExecBindParameter{
				Type:  goTypeToSnowflake(parameters[i]),
				Value: v1,
			}
		}
	}
	params := &url.Values{} // TODO: delete?

	headers := make(map[string]string)
	headers["Content-Type"] = ContentTypeApplicationJson
	headers["accept"] = AcceptTypeAppliationSnowflake // TODO: change to JSON in case of PUT/GET
	headers["User-Agent"] = UserAgent

	if sc.rest.Token != "" {
		headers[HeaderAuthorizationKey] = fmt.Sprintf(HeaderSnowflakeToken, sc.rest.Token)
	}

	var json_body []byte
	json_body, err = json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var data *ExecResponse
	data, err = sc.rest.PostQuery(params, headers, json_body, sc.rest.RequestTimeout)
	if err != nil {
		return nil, err
	}
	var code int
	if data.Code != "" {
		code, err = strconv.Atoi(data.Code)
		if err != nil {
			code = -1
			return nil, err
		}
	} else {
		code = -1
	}
	log.Printf("Success: %v, Code: %v", data.Success, code)
	if !data.Success {
		return nil, &SnowflakeError{
			Number:   code,
			SqlState: data.Data.SqlState,
			Message:  data.Message,
			QueryId:  data.Data.QueryId,
		}
	} else {
		log.Printf("Exec/Query SUCCESS: %v")
		sc.cfg.Database = data.Data.FinalDatabaseName
		sc.cfg.Schema = data.Data.FinalSchemaName
		sc.cfg.Role = data.Data.FinalRoleName
		sc.cfg.Warehouse = data.Data.FinalWarehouseName
		sc.QueryId = data.Data.QueryId
		sc.SqlState = data.Data.SqlState
	}
	return data, err
}

func (sc *snowflakeConn) Begin() (driver.Tx, error) {
	log.Println("Begin")
	_, err:= sc.exec("BEGIN", false, false, nil)
	if err != err {
		return nil, err
	}
	return &snowflakeTx{sc}, err
}
func (sc *snowflakeConn) Close() (err error) {
	log.Println("Close")
	// TODO
	return nil
}
func (sc *snowflakeConn) Prepare(query string) (driver.Stmt, error) {
	log.Println("Prepare")
	stmt := &snowflakeStmt{
		sc:    sc,
		query: query,
	}
	return stmt, nil
}
func (sc *snowflakeConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	log.Printf("Exec: %#v, %v", query, args)
	// TODO: handle noResult and isInternal
	data, err := sc.exec(query, false, false, args)
	if err != nil {
		return nil, err
	}
	var updatedRows int64
	if sc.isDml(data.Data.StatementTypeId) {
		// collects all values from the returned row sets
		updatedRows = 0
		for i, n := 0, len(data.Data.RowType); i < n; i++ {
			v, err := strconv.ParseInt(*data.Data.RowSet[0][i], 10, 64)
			if err != nil {
				return nil, err
			}
			updatedRows += v
		}
	}
	log.Printf("number of updated rows: %#v", updatedRows)
	return &snowflakeResult{
		affectedRows: updatedRows,
		insertId:     -1}, nil // last insert id is not supported by Snowflake
}

func (sc *snowflakeConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	log.Println("Query")
	// TODO: handle noResult and isInternal
	data, err := sc.exec(query, false, false, args)
	if err != nil {
		log.Printf("You got error: %v", err)
		return nil, err
	}

	rows := new(snowflakeRows)
	rows.sc = sc
	rows.RowType = data.Data.RowType
	rows.ChunkDownloader = &snowflakeChunkDownloader{
		CurrentChunk:  data.Data.RowSet,
		ChunkMetas:    data.Data.Chunks,
		Total:         int64(data.Data.Total),
		TotalRowIndex: int64(-1),
		Qrmk:          data.Data.Qrmk,
	}
	rows.ChunkDownloader.Start()
	return rows, err
}
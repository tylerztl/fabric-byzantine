package mysql

import (
	"database/sql"
	"encoding/json"
	"fabric-byzantine/server/helpers"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type DBMgr struct {
	db        *sql.DB
	stmtTx    *sql.Stmt
	stmtBlock *sql.Stmt
}

var dbConf = helpers.GetAppConf().Conf.DB
var dbMgr = NewDBMgr()

func NewDBMgr() *DBMgr {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.Name))
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	db.SetMaxOpenConns(dbConf.MaxOpenConns)
	db.SetMaxIdleConns(dbConf.MaxIdleConns)
	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	stmtBlock, err := db.Prepare(blockSQL) // ? = placeholder
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	stmtTx, err := db.Prepare(txSQL) // ? = placeholder
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return &DBMgr{db, stmtTx, stmtBlock}
}

func GetDBMgr() *DBMgr {
	return dbMgr
}

func (m *DBMgr) QueryRows(query string, args ...interface{}) ([]byte, error) {
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	datas := make([]map[string]string, 0)
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		entry := make(map[string]string)
		var value string
		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			entry[columns[i]] = value
		}
		datas = append(datas, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return json.Marshal(datas)
}

func (m *DBMgr) GetBlockHeight() uint64 {
	rows, err := m.db.Query("select max(number) as height from block")
	if err != nil {
		panic(err.Error())
	}
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error())
	}
	if len(columns) != 1 {
		panic("GetBlockHeight invalid height.")
	}
	for rows.Next() {
		var col uint64
		err = rows.Scan(&col)
		if err != nil {
			return 0
		}
		return col
	}
	return 0
}

func CloseDB() {
	if err := dbMgr.db.Close(); err != nil {
		panic(err)
	}
	if err := dbMgr.stmtTx.Close(); err != nil {
		panic(err)
	}
	if err := dbMgr.stmtBlock.Close(); err != nil {
		panic(err)
	}
}

func GetDB() *sql.DB {
	return dbMgr.db
}

func GetStmtTx() *sql.Stmt {
	return dbMgr.stmtTx
}

func GetStmtBlock() *sql.Stmt {
	return dbMgr.stmtBlock
}

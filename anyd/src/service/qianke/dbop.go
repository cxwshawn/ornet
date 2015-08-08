package qianke

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"service/dlog"
)

type QkDBOp struct {
	db  *sql.DB
	err error
}

func (op *QkDBOp) Init(mysqlHost, mysqlUsername, mysqlPasswd, dbName string) error {
	op.db, op.err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s",
		mysqlUsername, mysqlPasswd, mysqlHost, dbName))
	if op.err != nil {
		dlog.EPrintln(op.err.Error())
		return op.err
	}
	op.err = op.db.Ping()
	if op.err != nil {
		dlog.EPrintln(op.err.Error())
		return op.err
	}
	return nil
}

func (op *QkDBOp) Result() error {
	return op.err
}

func (op *QkDBOp) RegisterUser(userInfo *QKUser) error {
	res, err := op.db.Exec(fmt.Sprintf(`insert into qkuser(email,passwd,cellphone) values('%s', '%s', '%s')`, userInfo.Email, userInfo.Password, userInfo.CellPhone))
	if err != nil {
		op.err = err
		dlog.EPrintln(err.Error())
		return err
	}
	rows, err := res.RowsAffected()
	if rows == 0 || err != nil {
		dlog.EPrintf("register new user failed", err.Error())
		return errors.New("register new user failed.")
	}
	return nil
}

func (op *QkDBOp) Uninit() {
	op.db.Close()
}

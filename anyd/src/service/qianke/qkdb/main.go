package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func main() {
	log.SetFlags(log.Llongfile)

	db, err := sql.Open("mysql", "root:d4cd390a@tcp(127.0.0.1:3306)/mysql")
	if err != nil {
		log.Println(err.Error())
		return
	}
	_, err = db.Exec("create database if not exists qianke ;")
	if err != nil {
		log.Println(err.Error())
		return
	}
	_, err = db.Exec("use qianke;")
	if err != nil {
		log.Println(err.Error())
		return
	}
	_, err = db.Exec("drop table if exists qkuser;")
	if err != nil {
		log.Println(err.Error())
		return
	}
	_, err = db.Exec("create table  qkuser ( email varchar(50), passwd varchar(50), cellphone char(11), nickname varchar(50),sex TINYINT(1), birth TIMESTAMP, job varchar(50), registerTime TIMESTAMP);")
	if err != nil {
		log.Println(err.Error())
		return
	}
	db.Close()
}

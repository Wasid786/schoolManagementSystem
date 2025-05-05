package internal

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func Dbconnect() *sql.DB {
	db, err := sql.Open("mysql", "new_user:sql@/schooldb?parseTime=true")

	if err != nil {
		panic(err.Error())

	}

	return db

}

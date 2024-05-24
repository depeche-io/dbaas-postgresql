package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

const (
	sleepMillis = 10
)

var myIdent = time.Now().UnixNano()
var myCounter = 0
var lastWrittenCount = 0

func main() {
	host := os.Getenv("HOST")
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	createTable(psqlInfo)

	if os.Getenv("MODE") == "single" {
		ethernalConnection(psqlInfo)
	} else {
		separateConnection(psqlInfo)
	}
}

func separateConnection(psqlInfo string) {

}

func ethernalConnection(psqlInfo string) {
	for {
		singleConnection(psqlInfo)
	}
}

func singleConnection(psqlInfo string) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for {
		sqlStatement := `
			INSERT INTO ledger(myident, mynumber, app_insert_timestamp)
			VALUES ($1, $2, $3)
		`
		myCounter += 1
		err = db.QueryRow(sqlStatement, myIdent, myCounter, time.Now()).Scan()
		if err != sql.ErrNoRows {
			fmt.Println(err)
			return
		} else {
			lastWrittenCount = myCounter
		}

		lastWritten := 0
		err = db.QueryRow("SELECT MAX(mynumber) FROM ledger WHERE myident = $1", myIdent).Scan(&lastWritten)
		if err != nil {
			fmt.Println(err)
			return
		}
		if lastWritten != lastWrittenCount {
			// TODO: check here
		}

		time.Sleep(sleepMillis * time.Millisecond)
	}
}

func createTable(psqlInfo string) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sqlStatement := `
		CREATE TABLE IF NOT EXISTS ledger(
			myident bigint not null,
			mynumber bigint not null,
			app_insert_timestamp timestamp not null,
			db_insert_timestamp timestamp default NOW(),
			primary key (myident, mynumber)
		);
	`
	err = db.QueryRow(sqlStatement).Scan()
	if err != sql.ErrNoRows {
		panic(err)
	}
}

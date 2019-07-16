package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"time"
)

// db stores the connection to the database
var db *sql.DB

// DSNGenerator returns DSN filed with values from environment
func DSNGenerator() string {
	db_user := "libredns"
	db_password := "libredns"
	db_host := ""
	db_db := "libredns"

	if v, exists := os.LookupEnv("MYSQL_HOST"); exists {
		db_host = v
	}
	if v, exists := os.LookupEnv("MYSQL_PASSWORD"); exists {
		db_password = v
	} else if v, exists := os.LookupEnv("MYSQL_ROOT_PASSWORD"); exists {
		db_user = "root"
		db_password = v
	}
	if v, exists := os.LookupEnv("MYSQL_USER"); exists {
		db_user = v
	}
	if v, exists := os.LookupEnv("MYSQL_DATABASE"); exists {
		db_db = v
	}

	return db_user + ":" + db_password + "@" + db_host + "/" + db_db
}

// DBInit establishes the connection to the database
func DBInit(dsn string) (err error) {
	if db, err = sql.Open("mysql", dsn+"?parseTime=true&foreign_key_checks=1"); err != nil {
		return
	}

	_, err = db.Exec(`SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO';`)
	for i := 0; err != nil && i < 45; i += 1 {
		if _, err = db.Exec(`SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO';`); err != nil && i <= 45 {
			log.Println("An error occurs when trying to connect to DB, will retry in 2 seconds: ", err)
			time.Sleep(2 * time.Second)
		}
	}

	return
}

// DBCreate creates all necessary tables used by the package
func DBCreate() error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users(
  id_user INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(255) NOT NULL,
  password BINARY(64) NOT NULL,
  email VARCHAR(255) NOT NULL
) DEFAULT CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`); err != nil {
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS zones(
  id_zone INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  domain VARCHAR(255) NOT NULL,
  server VARCHAR(255),
  key_name VARCHAR(255) NOT NULL,
  key_blob BLOB NOT NULL
) DEFAULT CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`); err != nil {
		return err
	}
	return nil
}

// DBClose closes the connection to the database
func DBClose() error {
	return db.Close()
}

func DBPrepare(query string) (*sql.Stmt, error) {
	return db.Prepare(query)
}

func DBQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func DBExec(query string, args ...interface{}) (sql.Result, error) {
	return db.Exec(query, args...)
}

func DBQueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRow(query, args...)
}

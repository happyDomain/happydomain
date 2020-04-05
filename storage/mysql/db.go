package database

import (
	"os"
)

// DSNGenerator returns DSN filed with values from environment
func DSNGenerator() string {
	db_user := "happydns"
	db_password := "happydns"
	db_host := ""
	db_db := "happydns"

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

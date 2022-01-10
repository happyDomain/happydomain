// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package database

import (
	"flag"
	"os"

	"git.happydns.org/happydomain/storage"
)

var dsn string

func init() {
	storage.StorageEngines["mysql"] = Instantiate

	flag.StringVar(&dsn, "mysql-dsn", DSNGenerator(), "DSN to connect to the MySQL server")
}

// DSNGenerator returns DSN filed with values from environment
func DSNGenerator() string {
	db_user := "happydomain"
	db_password := "happydomain"
	db_host := ""
	db_db := "happydomain"

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

func Instantiate() (storage.Storage, error) {
	return NewMySQLStorage(dsn)
}

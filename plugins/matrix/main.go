package main

import (
	"git.happydns.org/happyDomain/model"
)

func NewTestPlugin() (happydns.TestPlugin, error) {
	return &MatrixTester{
		TesterURI: "https://federationtester.matrix.org/api/report?server_name=%s",
	}, nil
}

package main

import (
	"git.happydns.org/happyDomain/model"
)

func NewCheckPlugin() (string, happydns.Checker, error) {
	return "matrixim", &MatrixTester{
		TesterURI: "https://federationtester.matrix.org/api/report?server_name=%s",
	}, nil
}

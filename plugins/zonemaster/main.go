package main

import (
	"git.happydns.org/happyDomain/model"
)

func NewTestPlugin() (happydns.TestPlugin, error) {
	return &ZonemasterTest{}, nil
}

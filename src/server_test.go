package main

import "testing"

func TestInit(t *testing.T) {
	loadConfig()
	ret := InitEnv("")
	if ret != 0 {
		t.Fatalf("Init nvwa environment failed")
	}
}

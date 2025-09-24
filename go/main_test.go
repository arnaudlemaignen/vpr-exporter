package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	log "github.com/sirupsen/logrus"
)

// Init
func init() {
	// see
	// https://stackoverflow.com/questions/23847003/golang-tests-and-working-directory
	_, filename, _, _ := runtime.Caller(0)
	// The ".." may change depending on you folder structure
	dir := path.Join(path.Dir(filename), "../test")
	fmt.Println(dir)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

// Main of the test to init global vars, prepare things
// https://medium.com/goingogo/why-use-testmain-for-testing-in-go-dafb52b406bc
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

// main.go
func TestReady(t *testing.T) {
	want := "VPR Exporter is ready to rock"
	if got := Ready(); got != want {
		t.Errorf("Ready() = %q, want %q", got, want)
	}
}

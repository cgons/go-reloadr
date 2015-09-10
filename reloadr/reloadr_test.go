package main

import (
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"gopkg.in/fsnotify.v1"
)

// Setup & Mocks
// ----------------------------------------------
var gopath = os.Getenv("GOPATH")

type mockResponder struct {
	changeDetected bool
}

func (mresp *mockResponder) onChange(r *reloadr, event fsnotify.Event) {
	mresp.changeDetected = true
}

func (mresp *mockResponder) onErr(r *reloadr, err error) {}

// Test Methods
// ----------------------------------------------
func TestWatcherDiscoversAllDirs(t *testing.T) {
	// Since removing a non-existant path from a Watcher will return an error, we iterate over our fixed dir tree
	// and remove dirs one-by-one. If all dirs are removed without error, we know the Watcher discovered all dirs
	// initially and thus, this test is successfull.
	baseFolder := "_testdata/watch_dir"
	dirTree := [4]string{
		baseFolder,
		baseFolder + "/dir1",
		baseFolder + "/dir1/dir2",
		baseFolder + "/dir1/dir2/dir3",
	}
	watcher := setupWatcher(baseFolder)
	defer watcher.Close()

	for _, path := range dirTree {
		if err := watcher.Remove(path); err != nil {
			t.Errorf("DIR: '%s' was not picked up by the Watcher.", path)
		}
	}
}

func TestReloadrDetectsChange(t *testing.T) {
	_detectChangeHelper("_testdata/watch_dir/test_file.txt", t)
}

func TestReloadrDetectsNestedChange(t *testing.T) {
	_detectChangeHelper("_testdata/watch_dir/dir1/dir2/dir3/test_file.txt", t)
}

func TestAppInstalls(t *testing.T) {
	// Tests to ensure that the user app is compiled & installed correct via the "go install" command
	// setup
	r := new(reloadr)
	r.stopCh = make(chan bool)
	r.appName = "testappnormal"
	binarypath := path.Join(gopath, "bin", r.appName)
	os.Stdout = nil // supress output

	// change Reloadr's cmd dir to that of the test binary
	r.cmdDir = path.Join("_testdata", "binary", "testappnormal")

	defer func() { // cleanup
		os.Remove(binarypath)
	}()

	// install the test app/binary
	err := r.installApplication()

	if err != nil {
		t.Error("Test application was not installed due to the following error:")
		t.Error(err)
	}

	// check that the binary is installed
	if _, err = os.Stat(binarypath); err != nil {
		t.Error("Test application was not installed -- test binary NOT found in Go's bin dir." +
			"No error was reported during Reloadr's app install method")
	}
}

// func TestAppRuns(t *testing.T) {
// 	// Tests to ensure that Reloadr is able to run the user app.
// 	// setup
// 	// origStdout := os.Stdout
// 	// os.Stdout = nil // supress Stdout
// 	// // copy testapp binary to Go's bin dir
// 	// src := path.Join("_testdata", "binary", "testappnormal", "testappnormal")
// 	// dest := path.Join(gopath, "bin")
// 	// err := exec.Command("cp", src, dest).Run()
// 	// if err != nil {
// 	// 	t.Skip("Testapp binary could not be copied to Go's bin dir.")
// 	// }
// 	//
// 	// r := new(reloadr)
// 	// r.stopCh = make(chan bool)
// 	// r.appName = "testappnormal"
// 	//
// 	// os.Stdout = origStdout
// }

// func TestAppnReloads(t *testing.T) {
//
// }
//
// func TestAppInstallErrOutputCaptured(t *testing.T) {
//
// }
//
// func TestAppOutputCaptured(t *testing.T) {
//
// }
//
// func TestAppErrorOutputCaptured(t *testing.T) {
//
// }

// func TestAppRunsOnStart(t *testing.T) {
// 	// Test to ensure that user app is compiled and runs on Reloadr start
// 	// setup
// 	// r := new(reloadr)
// 	// r.init()
// 	// r.appName = "testappnormal" // overide some properties
// 	// r.watchPath = "_testdata/binary/testappnormal"
// 	// r.cmdDir = r.watchPath
// 	// r.resp = new(mockResponder)
// 	//
// 	// defer func() { // cleanup
// 	// 	_pkill(r.appName)
// 	// }()
// 	//
// 	// r.installAndRunApplication()
// 	// go r.watch()
// }

// Helpers
// ----------------------------------------------
func _isProcessRunning(name string) bool {
	output, err := exec.Command("pgrep", name).Output()
	if err != nil || len(output) == 0 {
		return false
	}
	return true
}

func _pkill(name string) {
	exec.Command("pkill", name).Run()
}

func _detectChangeHelper(filepath string, t *testing.T) {
	// Setup testdata
	testFile := filepath
	os.Create(testFile)

	defer func() { // cleanup
		os.Remove(testFile)
	}()

	r := new(reloadr)
	r.stopCh = make(chan bool)
	r.watchExts = []string{".txt"}
	r.watchPath = "_testdata/watch_dir"
	r.resp = new(mockResponder)
	go r.watch()

	time.Sleep(100 * time.Millisecond) // pause to allow the watcher a chance to start up

	// Trigger the watcher by writing to our test file
	f, _ := os.OpenFile(testFile, os.O_WRONLY, 0666)
	f.WriteString("new")
	f.Close()

	time.Sleep(100 * time.Millisecond) // pause to allow our changed to be detected

	close(r.stopCh)
	if r.resp.(*mockResponder).changeDetected != true {
		t.Error("Watcher did not detect change")
	}

}

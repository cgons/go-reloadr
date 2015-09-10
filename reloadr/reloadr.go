package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/fsnotify.v1"
)

const (
	watchPath = "."

	// Various error messages
	killInstanceErrMsg = "Reloadr: ERROR - Unable to kill the previous instance of application ( %s ). " +
		"Please manually kill the application insance and try agian. Reloadr has terminated as a result.\n"
	buildInstallErrMsg = "Reloadr: ERROR - Unable to build/install application: ( %s )\n"
	startAppErrMsg     = "Reloadr: ERROR - Unable to start application: ( %s )\n"
)

type reloadr struct {
	appName    string
	watchPath  string
	watchExts  []string
	cmdDir     string
	cmd        *exec.Cmd
	stopCh     chan bool
	terminated bool
	resp       watchResponder
}

type watchResponder interface {
	onChange(*reloadr, fsnotify.Event)
	onErr(*reloadr, error)
}

type responder struct{}

func (resp *responder) onChange(r *reloadr, event fsnotify.Event) {
	fmt.Println("--------------------------------------------------")
	fmt.Println("Reloadr: Change detected -->", event.Name) // filename/path
	r.installAndRunApplication()
	time.Sleep(25 * time.Millisecond)
	if !r.terminated {
		fmt.Println("Reloadr: Watching for changes...")
	}
}

func (resp *responder) onErr(r *reloadr, err error) {
	fmt.Printf("Reloadr: ERROR - Detecting file changes: %s. Reloader has now terminated.\n", err)
	r.terminated = true
	close(r.stopCh)
}

// Reloadr Class
// --------------------------------------------------------------------------------------------------

func newReloadr() *reloadr {
	r := new(reloadr)
	r.init()
	return r
}

func (r *reloadr) init() {
	r.appName = getAppName()
	r.watchPath = "."
	r.watchExts = []string{".go", ".html", ".tpl", ".tmpl"}
	r.stopCh = make(chan bool)
	if r.resp == nil {
		r.resp = new(responder)
	}
}

func (r *reloadr) start() {
	go r.watch()

	fmt.Println("")
	fmt.Println("-- RELOADR v0.1.0 --")
	fmt.Println("--------------------------------------------------")
	fmt.Println("Reloadr: Running and watching for changes...")
	fmt.Println("         On:", r.watchExts, "files.")
	fmt.Println("--------------------------------------------------")

	r.installAndRunApplication()
	<-r.stopCh
}

func (r *reloadr) watch() {
	watcher := setupWatcher(r.watchPath)
	defer watcher.Close()

	var lastBuildTime time.Time
	for {
		select {
		case event := <-watcher.Events:
			// We are only concerned with file modifications (WRITES)
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Now we check if the modified file is of an extension that we are watching for
				for _, ext := range r.watchExts {
					if strings.HasSuffix(event.Name, ext) { // Our file mataches a watched extension
						// Due to possible event duplication, we check to ensure that we haven't received a build
						// notification within the last 500 Millisecond (as duplicated events are instantaneous)
						elapsedTime := time.Now().Sub(lastBuildTime)
						if elapsedTime > (time.Millisecond * 250) {
							// Last build a while ago. Lets build again...
							// Notify the user that we've detected a change and will begin rebuilding
							r.resp.onChange(r, event)
						}
						lastBuildTime = time.Now()
						break // file matched a watched ext and build completed ok - no need to check other exts
					}
				}
			}
		case err := <-watcher.Errors:
			r.resp.onErr(r, err)
		case <-r.stopCh:
			return
		}
	}
}

func (r *reloadr) installApplication() error {
	cmd := exec.Command("go", "install")
	cmd.Dir = r.cmdDir

	cmdErrOutputPipe, _ := cmd.StderrPipe()

	fmt.Println("Reloadr: Building application...")

	err := cmd.Start()
	if err != nil {
		fmt.Println(`Reloadr: ERROR - Unable to execute "go install". Reloadr will now terminate.`)
		r.terminated = true
		close(r.stopCh)
		return err
	}

	errOutput, _ := ioutil.ReadAll(cmdErrOutputPipe)

	err = cmd.Wait()
	if err != nil {
		fmt.Printf(buildInstallErrMsg, r.appName)
		fmt.Println(r.appName+":", string(errOutput))
		return err
	}
	fmt.Println("Reloadr: Done...")
	return nil
}

func (r *reloadr) runApplication() {
	r.cmd = exec.Command(r.appName)

	cmdErrOutput, _ := r.cmd.StderrPipe()
	cmdOutput, _ := r.cmd.StdoutPipe()

	scanner := bufio.NewScanner(cmdOutput)
	go func() {
		for scanner.Scan() {
			fmt.Printf("%s: %s\n", r.appName, scanner.Text())
		}
	}()

	err := r.cmd.Start()
	if err != nil {
		fmt.Printf(startAppErrMsg, r.appName)
	}

	// Stream error output from the application (in a new goroutine so we don't block)
	var errOutput []byte
	go func(eo []byte) {
		errOutput, _ = ioutil.ReadAll(cmdErrOutput)
		if len(errOutput) > 0 {
			fmt.Printf(startAppErrMsg, r.appName)
			errOutputStr := strings.TrimSpace(string(errOutput))
			errOutputSlice := strings.Split(errOutputStr, "\n")
			for _, line := range errOutputSlice {
				fmt.Printf("%s: %s\n", r.appName, line)
			}
		}
	}(errOutput)

	fmt.Println("Reloadr: (", r.appName, ") - Started and running...")

	// Wait for the application to finish.
	// (We ignore error messages from wait - unless the server shuts down gracefully, we'd have false positives)
	r.cmd.Wait()
}

func (r *reloadr) installAndRunApplication() {
	err := r.installApplication()

	// Before we run the updated application, we kill any previously running instance (if any)
	r.killApplicationInstance()

	// We check to ensure that building/install the app did not fail and only then attempt to run the app.
	if err == nil {
		go r.runApplication()
	}
}

func (r *reloadr) killApplicationInstance() {
	// We only attempt kill an app instance if we can get a ref to it and is still running
	// Note: "Process" is not nill for a process that has ended. ie. It does not indicate a running process.
	// Thus, we have to check that ProcessState is nill to ensure that we are only attempting to kill a running
	// process as ProcessState is only populated once a process has ended.
	if r.cmd != nil {
		if r.cmd.Process != nil && r.cmd.ProcessState == nil {
			err := r.cmd.Process.Kill()
			if err != nil {
				fmt.Printf(killInstanceErrMsg, r.appName)
				r.terminated = true
				close(r.stopCh)
			}
		}
	}
}

// Helper functions
// ----------------------------------------------
func getAppName() string {
	// Get & set the current directory name as our app's name.
	// Why? Each Go built executable is named after it's source directory.
	wd, _ := os.Getwd()
	return filepath.Base(wd)
}

func setupWatcher(path string) *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// Walk the current file tree and add each (sub)directory to the watcher
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				log.Println(err)
			}
		}
		return err
	})

	return watcher
}

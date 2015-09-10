# Go-Reloadr
Go application to reload (rebuild/install and run) Go applications when source code changes are detected.


**Latest version: v0.1.0**

## Features
- Supports watching multiple file extensions (currently: .go .html .tpl .tmlp)
- File watcher is recursive. Start `reloadr` at your project's root and it will detect changes to any nested file(s).
- Guards against duplicate/successive file change (events) so your app only reloads/rebuilds once per change.
- `reloadr` passes all output to the command line, so you may freely log/print to stdout from your application.
- Multi-platform support via fsnotify (https://github.com/go-fsnotify/fsnotify)

#### Coming Soon
- [ ] Specify file extensions to watch
- [ ] Specify folders/files to ignore
- Configure Go-Reloadr options via:
    - [ ] command line args
    - [ ] a config file (`.reloadr.conf`)
- [ ] Support passing args/flags to your application executable
- [ ] Colourize `reloadr` output
- [ ] Complete test suite & imporoved coverage

## Installation
To install, enter the following in your terminal of choice:

`go get github.com/cgons/go-reloadr/reloadr`

After executing the above, Go will have placed a binary file (`reloadr`) in the "bin" directory of your $GOPATH

Tip: For convenience, consider adding $GOPATH/bin to your $PATH

## Usage
Simply run `reloadr` in your project's root directory. You should see similar output:

```
$ reloadr

-- reloadr vX.X.X --
--------------------------------------------------
reloadr: Running and watching for changes...
         On: [.go .html .tpl .tmpl] files.
--------------------------------------------------
reloadr: Building application...
reloadr: Done...
reloadr: ( your_app ) - Started and running...
```
<small>
Note: `reloadr` uses `go install` to build & install your app. As a result, `go` must be on your PATH and GOPATH must be set as well.
</small>

<small>
See: [Golang.org - Writing Go Code](https://golang.org/doc/code.html) and [Go Wiki - GOPATH](https://github.com/golang/go/wiki/GOPATH) for more information.
</small>

## Similar Projects
- Gin (https://github.com/codegangsta/gin)
- Fresh (https://github.com/pilu/fresh)
- ComplieDaemon (https://github.com/githubnemo/CompileDaemon)
- rerun (https://github.com/skelterjohn/rerun)
- go-reload (bash script) (https://github.com/alexedwards/go-reload)

## Bugs, Issues & Contributing
Please post any issues, bugs, or feature requests on Github. All feedback is more than welcome.

---
_Go-Reloadr is MIT Licenced_

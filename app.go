package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

type metaFile struct {
	Id       string `yaml:"hash"`
	Url      string `yaml:"url"`
	Accesses int    `yaml:"accesses"`
}

var config = NewConfig()
var quit chan struct{}

func init() {
	quit = make(chan struct{})
}

func main() {
	config.loadConfig()
	go background()

	startWebServer()

	close(quit)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}

func startWebServer() {
	staticDir := config.StaticDir
	fileServer := http.FileServer(http.Dir(staticDir))

	r := mux.NewRouter()
	r.Handle("/", fileServer)
	r.HandleFunc("/{path}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["path"]
		http.Redirect(w, r, "/"+hash+"/", http.StatusSeeOther)
	})
	r.HandleFunc("/{path}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["path"]
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Hash", hash)

		http.ServeFile(w, r, staticDir+"/serve.html")
	})
	r.HandleFunc("/{path}/api", ApiRequest)
	r.HandleFunc("/{path}/view", ViewRequest)
	r.HandleFunc("/{path}/download", DownloadRequest)
	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	var wait = time.Second * 15
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGKILL
	// SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Error("Server Shutdown:", err)
		return
	}
}

func background() {
	done := true
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Println("Caught a [CTRL]+[C], stopping watch process ...")
			log.Debug(sig)

			done = false
			log.Info("Stopped...")
		}
	}()

	cleanup()

	go watch(config.OriginalPath, &done, addMetaData, removeMetaData, renameMetaData, func() {
		log.Fatal("Error while watching the file. Please check the file permissions and try again.")
	})

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				cleanup()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
func cleanup() {
	loopThroughFiles(config.OriginalPath, removeMetaData, false)
	loopThroughFiles(config.OriginalPath, addMetaData, false)
}

func watch(path string, done *bool, callback func(name string), removeCallback func(name string), renameCallback func(oldPath string, newPath string), errorCallback func()) {
	w := watcher.New()
	w.IgnoreHiddenFiles(true)
	w.FilterOps(watcher.Create, watcher.Remove, watcher.Rename, watcher.Move, watcher.Write)

	log.Info("Adding background watcher for changed files...")
	go func() {
		for *done {

			select {
			case event := <-w.Event:
				switch event.Op {
				case watcher.Create:
					if strings.HasSuffix(event.Path, ".meta") {
						continue
					}
					if PathExists(event.Path) {
						callback(event.Path)
					}
				case watcher.Remove:
					if strings.HasSuffix(event.Path, ".meta") {
						if !PathExists(strings.TrimSuffix(event.Path, ".meta")) {
							continue
						}
						callback(strings.TrimSuffix(event.Path, ".meta"))
					} else {
						removeCallback(event.Path + ".meta")
					}
				case watcher.Rename:
				case watcher.Move:
					renameCallback(event.OldPath, event.Path)
				case watcher.Write:
					go cleanup()
				}

			case err := <-w.Error:
				if err != nil {
					log.Println("error:", err)
				}
				go errorCallback()
				w.Close()
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.Add(path); err != nil {
		log.Fatal("Add failed:", err)
	}

	go func() {
		if err := w.Start(time.Second * 1); err != nil {
			log.Fatalln(err)
		}
	}()
}

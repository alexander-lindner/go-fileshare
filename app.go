package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/raahii/kutt-go"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type metaFile struct {
	Id       string `yaml:"hash"`
	Url      string `yaml:"url"`
	Accesses int    `yaml:"accesses"`
}

var config = NewConfig()

func main() {
	config.loadConfig()
	go background()

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
	r.HandleFunc("/{path}/api", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["path"]
		if hash == "" {
			http.Error(w, "No hash provided", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var err error
		if _, ok := config.data[hash]; ok {
			path := config.data[hash].(string)
			//metaFileName := path + ".meta"
			//metaFile := loadMetaFile(metaFileName)
			stat, err := os.Stat(path)
			if err != nil {
				http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
				return
			}
			pathName := stat.Name()
			size := strconv.FormatInt(stat.Size(), 10)
			_, err = w.Write([]byte("{\"name\":\"" + pathName + "\",\"size\":\"" + size + "\",\"hash\":\"" + hash + "\"}"))
		} else {
			_, err = w.Write([]byte("{\"hash\":\"" + hash + "\"}"))
		}
		if err != nil {
			log.Error("Couldn't write to the response writer. ", err)
			return
		}
	})
	r.HandleFunc("/{path}/view", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["path"]
		if _, ok := config.data[hash]; ok {
			path := config.data[hash].(string)
			content, err := ioutil.ReadFile(path)
			if err != nil {
				http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
				return
			}
			_, err = w.Write(content)
			if err != nil {
				log.Error("Couldn't write to the response writer. ", err)
				return
			}

			go func() {
				metaFileName := path + ".meta"
				metaFile := loadMetaFile(metaFileName)
				metaFile.Accesses++
				saveMetaFile(metaFileName, metaFile)
			}()

		}
	})
	r.HandleFunc("/{path}/download", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		hash := vars["path"]
		if _, ok := config.data[hash]; ok {
			path := config.data[hash].(string)
			stat, err := os.Stat(path)
			if err != nil {
				http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
				return
			}
			filename := stat.Name()
			w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
			w.Header().Set("Content-Type", "application/octet-stream")

			go func() {
				metaFileName := path + ".meta"
				metaFile := loadMetaFile(metaFileName)
				metaFile.Accesses++
				saveMetaFile(metaFileName, metaFile)
			}()

			http.ServeFile(w, r, path)
		}
	})
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
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
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

	loopThroughFiles(config.OriginalPath, addMetaData)
	go watch(config.OriginalPath, &done, addMetaData, func() {
		log.Fatal("Error while watching the file. Please check the file permissions and try again.")
	})
}

func addMetaData(name string) {
	newPath := config.OriginalPath + "/" + name
	if PathExists(newPath) && !IsDirectory(newPath) && filenameNotEndingWith(name, ".meta") && name[0] != '.' {
		metaFileName := newPath + ".meta"
		if !PathExists(metaFileName) {
			log.Info("File " + newPath + " has been changed, but no meta file found. Creating...")
			randomString := RandStringBytesMaskImprSrcUnsafe(config.hashSize)
			fileContent := metaFile{
				Accesses: 0,
				Id:       randomString,
				Url:      config.BaseUrl + "/" + randomString,
			}

			config.data[randomString] = newPath

			if config.Kutt.IsUrlShortenerEnabled {
				cli := kutt.NewClient(config.Kutt.UrlShortenerApiKey)
				cli.BaseURL = config.Kutt.UrlShortenerUrl
				URL, err := cli.Submit(
					fileContent.Url,
				)
				if err != nil {
					log.Error("Error while creating the url shortener. ", err)
				}
				fileContent.Url = URL.ShortURL
			}
			saveMetaFile(metaFileName, fileContent)

			go config.saveData()
		}
	}
}

func loadMetaFile(metaFileName string) metaFile {
	content, err := ioutil.ReadFile(metaFileName)
	if err != nil {
		log.Error("Couldn't read from "+metaFileName, err)
	}
	var fileContent metaFile
	err = yaml.Unmarshal(content, &fileContent)
	if err != nil {
		log.Error("Couldn't unyamlize the meta file. ", err)
	}
	return fileContent
}

func saveMetaFile(metaFileName string, fileContent metaFile) {
	content, err := yaml.Marshal(&fileContent)
	if err != nil {
		log.Panic("Couldn't yamlize the meta file. ", err)
	}
	err = os.WriteFile(metaFileName, content, 0644)
	if err != nil {
		log.Panic("Couldn't write to "+metaFileName, err)
	}
}

func watch(path string, done *bool, callback func(name string), errorCallback func()) {
	w := watcher.New()
	w.FilterOps(watcher.Write, watcher.Create)

	log.Info("Adding background watcher for changed files...")
	go func() {
		for *done {

			select {
			case event := <-w.Event:
				callback(event.Name())

			case err := <-w.Error:
				if err != nil {
					log.Println("error:", err)
				}
				go func() {
					errorCallback()
				}()
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

package main

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func ApiRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["path"]
	if hash == "" {
		http.Error(w, "No hash provided", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var err error
	if hashExists(hash) {
		path := GetPathFromHash(hash)
		stat, err := os.Stat(path)
		if err != nil {
			http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
			return
		}
		pathName := stat.Name()
		typeOfFile := "file"
		var size int64
		if stat.IsDir() {
			size, err = DirSize(path)
			if err != nil {
				http.Error(w, "Couldn't detect size for "+path, http.StatusInternalServerError)
				return
			}
			pathName += ".tar.gz"
			typeOfFile = "dir"
		} else {
			size = stat.Size()
		}
		_, err = w.Write([]byte("{\"name\":\"" + pathName + "\",\"size\":\"" + strconv.FormatInt(size, 10) + "\",\"hash\":\"" + hash + "\", \"type\":\"" + typeOfFile + "\"}"))
	} else {
		_, err = w.Write([]byte("{\"hash\":\"" + hash + "\"}"))
	}
	if err != nil {
		log.Error("Couldn't write to the response writer. ", err)
		return
	}
}
func ViewRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["path"]
	if hashExists(hash) {
		path := GetPathFromHash(hash)
		stat, err := os.Stat(path)
		if !PathExists(path) {
			http.Error(w, "Cannot read path: "+path, http.StatusInternalServerError)
			return
		}
		var content []byte
		if err == nil && stat.IsDir() {
			var files []string
			loopThroughFiles(path, func(p string) {
				files = append(files, "<li>"+strings.Replace(p, path, "", 1)+"</li>")
			}, true)
			content = []byte("<p>Directory content:</p><ul>" + strings.Join(files, "") + "</ul>")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			content, err = ioutil.ReadFile(path)
			if err != nil {
				http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
				return
			}
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
}
func DownloadRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["path"]
	if hashExists(hash) {
		path := GetPathFromHash(hash)
		stat, err := os.Stat(path)
		if err != nil {
			http.Error(w, "Couldn't read from "+path, http.StatusInternalServerError)
			return
		}
		filename := stat.Name()
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename)+".tar.gz")
		w.Header().Set("Content-Type", "application/octet-stream")

		go func() {
			metaFileName := path + ".meta"
			metaFile := loadMetaFile(metaFileName)
			metaFile.Accesses++
			saveMetaFile(metaFileName, metaFile)
		}()
		if stat, err = os.Stat(path); err == nil && stat.IsDir() {
			tar, _ := tarIt(path)
			_, err := w.Write(tar.Bytes())
			if err != nil {
				http.Error(w, "Cannot download tar.gz file: "+path, http.StatusInternalServerError)
			}
		} else {
			http.ServeFile(w, r, path)
		}
	}
	return
}

func GetPathFromHash(hash string) string {
	returnValue := ""
	loopThroughFiles(config.OriginalPath, func(path string) {
		if !strings.HasSuffix(path, ".meta") {
			return
		}
		meta := loadMetaFile(path)
		if meta.Id == hash {
			returnValue = strings.TrimSuffix(path, ".meta")
		}
	}, false)
	return returnValue
}

func hashExists(hash string) bool {
	returnValue := false
	loopThroughFiles(config.OriginalPath, func(path string) {
		if !strings.HasSuffix(path, ".meta") {
			return
		}
		meta := loadMetaFile(path)
		if meta.Id == hash {
			returnValue = PathExists(strings.TrimSuffix(path, ".meta"))
		}
	}, false)
	return returnValue
}

package main

import (
	"github.com/raahii/kutt-go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
)

func saveMetaFile(metaFileName string, fileContent metaFile) {
	if PathExists(metaFileName) {
		log.Warn("Meta file already exists. Skipping...")
		return
	}
	content, err := yaml.Marshal(&fileContent)
	if err != nil {
		log.Panic("Couldn't yamlize the meta file. ", err)
	}
	err = os.WriteFile(metaFileName, content, 0644)
	if err != nil {
		log.Panic("Couldn't write to "+metaFileName, err)
	}
}
func loadMetaFile(metaFileName string) (fileContent metaFile) {
	if !PathExists(metaFileName) {
		log.Error("Couldn't find the file: ", metaFileName)
		return
	}
	content, err := ioutil.ReadFile(metaFileName)
	if err != nil {
		log.Error("Couldn't read from "+metaFileName, err)
		return
	}

	err = yaml.Unmarshal(content, &fileContent)
	if err != nil {
		log.Error("Couldn't unyamlize the meta file. ", err)
		return
	}
	return
}
func addMetaData(name string) {
	newPath := name
	metaFileName := newPath + ".meta"
	if !PathExists(newPath) {
		log.Error("File doesn't exist: ", newPath)
		return
	}
	if strings.HasSuffix(name, ".meta") {
		log.Debug("Skipping meta file: ", newPath)
		return
	}
	if name[0] == '.' {
		log.Debug("Skipping hidden file: ", newPath)
		return
	}
	if PathExists(metaFileName) {
		log.Debug("Meta file already exists: ", metaFileName)
		return
	}

	log.Info("File " + newPath + " has been changed, but no meta file found. Creating...")
	randomString := RandStringBytesMaskImprSrcUnsafe(config.HashSize)
	fileContent := metaFile{
		Accesses: 0,
		Id:       randomString,
		Url:      config.BaseUrl + "/" + randomString,
	}

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
}

func removeMetaData(filePath string) {
	if !strings.HasSuffix(filePath, ".meta") {
		return
	}
	if PathExists(strings.TrimSuffix(filePath, ".meta")) {
		return
	}
	log.Info("Cleaned meta file: ", filePath)
	err := os.Remove(filePath)
	if err != nil {
		log.Error("Error while removing file: ", err)
		return
	}
}

func renameMetaData(oldPath string, newPath string) {
	if !PathExists(oldPath) {
		log.Error("Couldn't find the file: ", oldPath, ". Skipping...")
		return
	}
	err := os.Rename(oldPath, newPath)
	if err != nil {
		log.Error("Error while renaming file: ", err)
		return
	}
}

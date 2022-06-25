package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	data map[string]interface{}

	OriginalPath     string `yaml:"path"`
	globalConfigFile string
	DataFile         string `default:"./data.yaml" yaml:"DataFile"`
	BaseUrl          string `default:"http://localhost:8080" yaml:"BaseUrl"`
	Kutt             struct {
		UrlShortenerApiKey    string `default:"" yaml:"key"`
		IsUrlShortenerEnabled bool   `default:"false" yaml:"enabled"`
		UrlShortenerUrl       string `default:"" yaml:"url"`
	}
	StaticDir string
	hashSize  int `default:"128" yaml:"hashLength"`
}

func init() {

}
func NewConfig() Config {
	var config Config
	config.init()
	return config
}
func (c *Config) init() {
	c.data = make(map[string]interface{})
	c.globalConfigFile = "./config.yaml"

	c.loadConfig()

	if PathExists("./static") {
		c.StaticDir = "./static"
	} else {
		c.StaticDir = "/static"
	}

	if c.OriginalPath == "" {
		_, ok := os.LookupEnv("IN_DOCKER")
		if ok {
			c.OriginalPath = "/data"
		} else {
			c.OriginalPath = "./"
		}
	}

	c.loadData()
}

func (c *Config) saveConfig() {
	content, err := yaml.Marshal(&c.data)
	if err != nil {
		log.Panic("Couldn't yamlize the config file. ", err)
	}
	err = os.WriteFile(c.globalConfigFile, content, 0644)
	if err != nil {
		log.Panic("Couldn't write to "+c.globalConfigFile, err)
	}
}
func (c *Config) loadConfig() {
	if PathExists(c.globalConfigFile) {
		content, err := os.ReadFile(c.globalConfigFile)
		if err != nil {
			log.Panic("Couldn't read from "+c.globalConfigFile, err)
		}
		err = yaml.Unmarshal(content, c)
		if err != nil {
			log.Panic("Couldn't read from "+c.globalConfigFile, err)
		}
	} else {
		log.Panic("Couldn't find " + c.globalConfigFile)
	}
}
func (c *Config) saveData() {
	content, err := yaml.Marshal(&c.data)
	if err != nil {
		log.Panic("Couldn't yamlize the config file. ", err)
	}
	err = os.WriteFile(c.DataFile, content, 0644)
	if err != nil {
		log.Panic("Couldn't write to "+c.DataFile, err)
	}
}
func (c *Config) loadData() {
	if PathExists(c.DataFile) {
		content, err := os.ReadFile(c.DataFile)
		if err != nil {
			log.Panic("Couldn't read from "+c.DataFile, err)
		}
		err = yaml.Unmarshal(content, c.data)
		if err != nil {
			log.Panic("Couldn't read from "+c.DataFile, err)
		}
	}
}

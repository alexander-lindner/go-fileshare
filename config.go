package main

import (
	"github.com/creasty/defaults"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	OriginalPath     string `yaml:"path"`
	globalConfigFile string
	BaseUrl          string `default:"http://localhost:8080" yaml:"BaseUrl"`
	Kutt             struct {
		UrlShortenerApiKey    string `default:"" yaml:"key"`
		IsUrlShortenerEnabled bool   `default:"false" yaml:"enabled"`
		UrlShortenerUrl       string `default:"" yaml:"url"`
	}
	StaticDir string
	HashSize  int `default:"128" yaml:"hashLength"`
}

func init() {

}
func NewConfig() Config {
	var config Config
	config.init()
	return config
}
func (c *Config) init() {
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
}

func (c *Config) saveConfig() {
	content, err := yaml.Marshal(&c)
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
		err = defaults.Set(c)
		if err != nil {
			log.Error("Couldn't set defaults for config. ", err)
			log.Println("Current config: ", c)
		}
	} else {
		log.Panic("Couldn't find " + c.globalConfigFile)
	}
}

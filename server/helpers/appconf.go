package helpers

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type AppConf struct {
	Conf Application `yaml:"application"`
}

type Application struct {
	LogPath       string          `yaml:"logPath"`
	LogLevel      int8            `yaml:"logLevel"`
	OrgInfo       []*OrgInfo      `yaml:"org"`
	OrderderInfo  []*OrderderInfo `yaml:"orderer"`
	DB            *DBInfo         `yaml:"db"`
	TxNumPerBlock int64           `yaml:"txNumPerBlock"`
	TxInterval    int64           `yaml:"txInterval"`
}

type OrgInfo struct {
	Name  string `yaml:"name"`
	Admin string `yaml:"admin"`
	User  string `yaml:"user"`
}

type OrderderInfo struct {
	Name     string `yaml:"name"`
	Admin    string `yaml:"admin"`
	Endpoint string `yaml:"endpoint"`
}

type DBInfo struct {
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Name         string `yaml:"name"`
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	MaxOpenConns int    `yaml:"maxOpenConns"`
	MaxIdleConns int    `yaml:"maxIdleConns"`
}

var appConfig = new(AppConf)

func init() {
	confPath := GetConfigPath("app.yaml")
	yamlFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(fmt.Errorf("yamlFile.Get err[%s]", err))
	}
	if err = yaml.Unmarshal(yamlFile, appConfig); err != nil {
		panic(fmt.Errorf("yamlFile.Unmarshal err[%s]", err))
	}
}

func GetAppConf() *AppConf {
	return appConfig
}

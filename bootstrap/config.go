package bootstrap

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/healer1219/martini/config"
	"github.com/healer1219/martini/global"
	"github.com/spf13/viper"
	"log"
	"os"
)

func InitConfig() *global.Application {
	configFile := "config.yaml"
	if envConfigFile := os.Getenv("CONFIG_FILE"); envConfigFile != "" {
		configFile = envConfigFile
	}
	log.Printf("initing config, use log file %s \n", configFile)
	v := InitConfigByName(configFile)
	global.App.ConfigViper = v
	log.Printf("init config complete \n")
	return global.App
}

func InitConfigByName(fileName string) *viper.Viper {
	v := viper.New()
	v.SetConfigFile(fileName)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("read config file failed: %s \n", err))
	}
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {
		UnmarshalConfig(v)
	})
	UnmarshalConfig(v)
	return v
}

func UnmarshalConfig(v *viper.Viper) {
	unmarshalErr := v.Unmarshal(&global.App.Config)
	if unmarshalErr != nil {
		fmt.Println(unmarshalErr)
	}
}

func GetConfig(configFileName string) (*viper.Viper, config.Config) {
	v := viper.New()
	v.SetConfigFile(configFileName)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("read config file failed: %s \n", err))
	}
	conf := config.Config{}
	v.Unmarshal(&conf)
	return v, conf
}

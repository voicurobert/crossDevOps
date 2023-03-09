package main

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func main() {
	exePath, _ := os.Executable()
	path := filepath.Dir(exePath)
	config, err := LoadConfig(path)
	if err != nil {
		fmt.Println("error loading config: ", err)
		panic(err)
	}

	config.RunActions()
}

func LoadConfig(path string) (config CROSSConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = viper.Unmarshal(&config)

	return
}

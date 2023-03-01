package main

import (
	"fmt"
	"github.com/spf13/viper"
)

func main() {
	config, err := LoadConfig(".")
	if err != nil {
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

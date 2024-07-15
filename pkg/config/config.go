package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func init() {
	viper.AutomaticEnv()
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType(
		"yaml",
	) // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

func Viper() *viper.Viper {
	return viper.GetViper()
}

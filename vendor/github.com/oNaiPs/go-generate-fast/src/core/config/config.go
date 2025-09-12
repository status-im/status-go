package config

import (
	"os"
	"path"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ConfigDir     string
	CacheDir      string
	Disable       bool
	ReadOnly      bool
	ReCache       bool
	ForceUseCache bool
	Debug         bool
}

var instance *Config

func Get() *Config {
	return instance
}

func Init() {
	if instance != nil {
		zap.S().Panic("Config was already initialized")
	}

	instance = &Config{}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		zap.S().Fatal("Error finding user config dir: ", err)
	}
	userConfigDir = path.Join(userConfigDir, "go-generate-fast")

	viper.SetEnvPrefix("GO_GENERATE_FAST")
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(userConfigDir)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error
		} else {
			zap.S().Errorf("Cannot parse config file: %s", err)
		}
	}

	viper.SetDefault("dir", userConfigDir)
	instance.ConfigDir = viper.GetString("dir")
	CreateDirIfNotExists(instance.ConfigDir)

	viper.SetDefault("cache_dir", path.Join(instance.ConfigDir, "cache"))
	instance.CacheDir = viper.GetString("cache_dir")
	CreateDirIfNotExists(instance.CacheDir)

	instance.Disable = viper.GetBool("disable")
	instance.ReadOnly = viper.GetBool("read_only")
	instance.ReCache = viper.GetBool("recache")
	instance.ForceUseCache = viper.GetBool("force_use_cache")
	instance.Debug = viper.GetBool("debug")
}

func CreateDirIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0700)
		if err != nil {
			zap.S().Fatal("Error creating config directory: ", err)
			return
		}
	}
}

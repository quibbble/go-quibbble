package config

import (
	"github.com/spf13/viper"
	"strings"
)

func NewConfig(fileName, prefix string, cfg interface{}) error {
	v := viper.New()
	v.SetConfigName(fileName)
	v.AddConfigPath("configs")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")
	v.SetEnvPrefix(prefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return err
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return err
	}
	return nil
}

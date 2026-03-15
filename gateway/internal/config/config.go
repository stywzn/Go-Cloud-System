package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type RouteConfig struct {
	PathPrefix string `yaml:"path_prefix"`
	TargetURL  string `yaml:"target_url"`
}

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`

	Routes []RouteConfig `yaml:"routes"`
}

func LoadConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析 YANL 失败 : %v", err)
	}
	return &cfg
}

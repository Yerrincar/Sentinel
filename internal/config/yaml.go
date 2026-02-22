package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type state int

const (
	running state = iota
	stopped
	error
	unknown
)

type Settings struct {
	Polling struct {
		Interval int `yaml:"Interval"`
	}
}

type ServiceDef struct {
	Id     string `yaml:"Id"`
	Name   string `yaml:"Name"`
	Docker struct {
		ContainerName string `yaml:"Container"`
	}
	TypeOfService string `yaml:"Type"`
	Url           string `yaml:"Url"`
}

type YamlConfig struct {
	Version  int          `yaml:"Version"`
	Settings Settings     `yaml:"Settings"`
	Services []ServiceDef `yaml:"Services"`
}

func (y *YamlConfig) ReadFromConfigFile() []ServiceDef {
	yamlFile, err := os.ReadFile("./internal/config/config.yaml")
	if err != nil {
		log.Printf("Error reading yaml config file: %v", err)
	}
	err = yaml.Unmarshal(yamlFile, y)
	if err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}
	return y.Services
}

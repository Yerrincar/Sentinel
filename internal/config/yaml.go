package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
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
	} `yaml:"Docker"`
	K8s struct {
		Context   string `yaml:"Context"`
		Namespace string `yaml:"Namespace"`
		Pod       string `yaml:"Pod"`
	} `yaml:"K8s"`
	Systemd struct {
		Unit string `yaml:"Unit"`
	} `yaml:"Systemd"`
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

func (y *YamlConfig) ServicesInfo() []string {
	info := y.ReadFromConfigFile()
	services := make([]string, 0)
	for _, i := range info {
		switch i.TypeOfService {
		case "docker":
			services = append(services, i.Id+"\n"+i.Name+"\n"+i.Docker.ContainerName+"\n"+i.Url)
		}
	}
	return services
}

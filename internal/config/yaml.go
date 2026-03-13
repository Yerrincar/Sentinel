package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Polling struct {
		Interval string `yaml:"Interval"`
	} `yaml:"Polling"`
	Workspace struct {
		Name string `yaml:"Name"`
	} `yaml:"Workspace"`
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

func (y *YamlConfig) WriteYamlConfigFile(name string) {
	filePath := "./internal/config/config.yaml"
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading yaml config file: %v", err)
		return
	}

	var root yaml.Node
	err = yaml.Unmarshal(yamlFile, &root)
	if err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}

	if len(root.Content) == 0 {
		log.Printf("Error updating yaml: empty document")
		return
	}
	doc := root.Content[0]
	settings := mapNodeValue(doc, "Settings")
	if settings == nil {
		log.Printf("Error updating yaml: missing Settings")
		return
	}

	workspace := mapNodeValue(settings, "Workspace")
	if workspace == nil {
		workspace = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		settings.Content = append(settings.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Workspace"},
			workspace,
		)
	}

	if !setMapScalar(workspace, "Name", name) {
		workspace.Content = append(workspace.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Name"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
		)
	}

	updatedData, err := yaml.Marshal(&root)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}

	err = os.WriteFile(filePath, updatedData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	y.Settings.Workspace.Name = name
}

func mapNodeValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]
		if k.Value == key {
			return v
		}
	}
	return nil
}

func setMapScalar(m *yaml.Node, key, value string) bool {
	if m == nil || m.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]
		if k.Value == key {
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
			return true
		}
	}
	return false
}

func (y *YamlConfig) Interval() time.Duration {
	interval, _ := time.ParseDuration(y.Settings.Polling.Interval)
	if interval <= 0 {
		return time.Second
	}
	return interval
}

func (y *YamlConfig) ServicesInfo() []string {
	services := make([]string, 0)
	for _, i := range y.Services {
		switch i.TypeOfService {
		case "docker":
			services = append(services, i.Id+"\n"+i.Name+"\n"+i.Docker.ContainerName+"\n"+i.Url)
		}
	}
	return services
}

package manifest

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"hover/utils"
	"os"
	"path/filepath"
)

type Queue struct {
	Memory      int      `yaml:"memory" json:"memory"`
	Timeout     int      `yaml:"timeout" json:"timeout"`
	Concurrency int      `yaml:"concurrency" json:"concurrency"`
	Tries       int      `yaml:"tries" json:"tries"`
	Backoff     string   `yaml:"backoff" json:"backoff"`
	Queues      []string `yaml:"queues" json:"queues"`
}

type Manifest struct {
	Name           string            `yaml:"name" json:"name"`
	AwsProfile     string            `yaml:"aws-profile" json:"aws-profile"`
	Region         string            `yaml:"region" json:"region"`
	Environment    map[string]string `yaml:"environment" json:"environment"`
	DeployCommands []string          `yaml:"deploy-commands" json:"deploy-commands"`
	Dockerfile     string            `yaml:"dockerfile" json:"dockerfile"`
	Auth           struct {
		LambdaRole string `yaml:"lambda-role" json:"lambda-role"`
		StackRole  string `yaml:"stack-role" json:"stack-role"`
	} `yaml:"auth" json:"auth"`
	VPC struct {
		SecurityGroups []string `yaml:"security-groups" json:"security-groups"`
		Subnets        []string `yaml:"subnets" json:"subnets"`
	} `yaml:"vpc" json:"vpc"`
	HTTP struct {
		Memory      int    `yaml:"memory" json:"memory"`
		Timeout     int    `yaml:"timeout" json:"timeout"`
		Warm        int    `yaml:"warm" json:"warm"`
		Concurrency int    `yaml:"concurrency" json:"concurrency"`
		Domains     string `yaml:"domains" json:"domains"`
		Certificate string `yaml:"certificate" json:"certificate"`
	} `yaml:"http" json:"http"`
	Cli struct {
		Memory      int `yaml:"memory" json:"memory"`
		Timeout     int `yaml:"timeout" json:"timeout"`
		Concurrency int `yaml:"concurrency" json:"concurrency"`
	} `yaml:"cli" json:"cli"`
	Queue        map[string]Queue `yaml:"queue" json:"queue"`
	BuildDetails struct {
		Id   string `yaml:"id" json:"id"`
		Hash string `yaml:"hash" json:"hash"`
		Time int64  `yaml:"time" json:"time"`
	} `yaml:"build_details" json:"build_details"`
}

func Get(alias string) (*Manifest, error) {
	path := filepath.Join(utils.Path.Hover, alias+".yml")
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read the manifest file at `%s`. Error: %w", path, err)
	}

	var manifest Manifest

	err = yaml.Unmarshal(file, &manifest)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the YAML manifest file. Error: %w", err)
	}

	return &manifest, nil
}

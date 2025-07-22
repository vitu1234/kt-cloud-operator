package httpapi

import (
	"k8s.io/client-go/tools/clientcmd/api"
)

type RawKubeConfig struct {
	Kind           string          `yaml:"kind"`
	ApiVersion     string          `yaml:"apiVersion"`
	Clusters       []NamedCluster  `yaml:"clusters"`
	AuthInfos      []NamedAuthInfo `yaml:"users"`
	Contexts       []NamedContext  `yaml:"contexts"`
	CurrentContext string          `yaml:"current-context"`
}

type NamedCluster struct {
	Name    string      `yaml:"name"`
	Cluster api.Cluster `yaml:"cluster"`
}

type NamedAuthInfo struct {
	Name string       `yaml:"name"`
	User api.AuthInfo `yaml:"user"`
}

type NamedContext struct {
	Name    string      `yaml:"name"`
	Context api.Context `yaml:"context"`
}

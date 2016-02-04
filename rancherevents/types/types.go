package util

import (
	"github.com/rancher/go-rancher/client"
)

type Service struct {
	Name              string `json:"name"`
	UUID              string `json:"uuid"`
	Kind              string `json:"kind"`
	Stack             Stack  `json:"environment"`
	SelectorContainer string `json:"selectorContainer"`
	SelectorLink      string `json:"selectorLink"`
	Data              Data   `json:"data"`
	Scale             int32  `json:"scale"`
}

type Data struct {
	Fields Fields `json:"fields"`
}

type Fields struct {
	LaunchConfig                  client.LaunchConfig    `json:"launchConfig"`
	SecondaryLaunchConfigs        []client.LaunchConfig  `json:"secondaryLaunchConfigs"`
	SessionAffinity               string                 `json:"SessionAffinity"`
	ClusterIP                     string                 `json:"vip"`
	Type                          string                 `json:"serviceType"`
	ExternalIPs                   []string               `json:"externalIpAddresses"`
	Ports                         []Port                 `json:"ports"`
	Labels                        map[string]interface{} `json:"labels"`
	ActiveDeadlineSeconds         int64                  `json:"activeDeadlineSeconds"`
	DnsPolicy                     string                 `json:"dnsPolicy"`
	HostIPC                       bool                   `json:"hostIPC"`
	NodeName                      string                 `json:"nodeName"`
	ServiceAccountName            string                 `json:"serviceAccountName"`
	TerminationGracePeriodSeconds int64                  `json:"terminationGracePeriodSeconds"`
	NodeSelector                  string                 `json:"nodeSelector"`
	ImagePullSecrets              []string               `json:"imagePullSecrets"`
}

type Port struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

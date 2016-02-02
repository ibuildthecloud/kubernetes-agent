package util

type Service struct {
	Name            string                 `json:"name"`
	UUID            string                 `json:"uuid"`
	Kind            string                 `json:"kind"`
	Stack           Stack                  `json:"environment"`
	Selector        string                 `json:"selector"`
	ExternalIPs     []string               `json:"externalIpAddresses"`
	Data            map[string]interface{} `json:"data"`
	LoadBalancerIP  string                 `json:"loadBalancerIP"`
	SessionAffinity string                 `json:"SessionAffinity"`
	ClusterIP       string                 `json:"clusterIP"`
	Type            string                 `json:"serviceType"`
}

type LaunchConfig struct {
	Ports []string
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

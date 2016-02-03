package util

type Service struct {
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
	Kind     string `json:"kind"`
	Stack    Stack  `json:"environment"`
	Selector string `json:"selectorContainer"`
	Data     Data   `json:"data"`
}

type Data struct {
	Fields Fields `json:"fields"`
}

type Fields struct {
	LaunchConfig    LaunchConfig `json:"launchConfig"`
	SessionAffinity string       `json:"SessionAffinity"`
	ClusterIP       string       `json:"vip"`
	Type            string       `json:"serviceType"`
	ExternalIPs     []string     `json:"externalIpAddresses"`
	Ports           []Port       `json:"ports"`
}

type Port struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
}

type LaunchConfig struct {
	Labels map[string]string `json:"labels"`
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

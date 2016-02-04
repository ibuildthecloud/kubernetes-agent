package util

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
	LaunchConfig                  LaunchConfig           `json:"launchConfig"`
	SessionAffinity               string                 `json:"SessionAffinity"`
	ClusterIP                     string                 `json:"vip"`
	Type                          string                 `json:"serviceType"`
	ExternalIPs                   []string               `json:"externalIpAddresses"`
	Ports                         []Port                 `json:"ports"`
	Labels                        map[string]interface{} `json:"labels"`
	ActiveDeadlineSeconds         int64                  `json:"activeDeadlineSeconds"`
	DnsPolicy                     string                 `json:"dnsPolicy"`
	HostIPC                       bool                   `json:"hostIPC"`
	HostNetwork                   bool                   `json:"hostNetwork"`
	HostPID                       bool                   `json:"hostPID"`
	NodeName                      string                 `json:"nodeName"`
	ServiceAccountName            string                 `json:"serviceAccountName"`
	TerminationGracePeriodSeconds int64                  `json:"terminationGracePeriodSeconds"`
}

type Port struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
}

type LaunchConfig struct {
	Name   string                 `json:"name"`
	Labels map[string]interface{} `json:"labels"`
	Image  string                 `json:"imageUuid"`
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

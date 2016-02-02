package util

type Service struct {
	Name            string                 `json:"name"`
	UUID            string                 `json:"uuid"`
	Kind            string                 `json:"kind"`
	Stack           Stack                  `json:"environment"`
	Selector        string                 `json:"selectorContainer"`
	Data            map[string]interface{} `json:"data"`
	SessionAffinity string                 `json:"SessionAffinity"`
	ClusterIP       string                 `json:"vip"`
	Type            string                 `json:"serviceType"`
}

type LaunchConfig struct {
	Ports []string
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

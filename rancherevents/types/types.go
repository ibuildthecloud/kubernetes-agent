package util

type Service struct {
	Name  string `json:"name"`
	UUID  string `json:"uuid"`
	Kind  string `json:"kind"`
	Stack Stack  `json:"environment"`
	Data  Data   `json:"data"`
}

type Data struct {
	Fields Fields `json:"fields"`
}

type Fields struct {
	Template        interface{} `json:"template"`
	ResourceVersion string      `json:"resourceVersion"`
}

type Stack struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

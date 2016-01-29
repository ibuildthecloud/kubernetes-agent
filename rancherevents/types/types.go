package util

type Service struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Stack Stack  `json:"environment"`
}

type Stack struct {
	Name string `json:"name"`
}

type KubenetesService struct {
	Name string `json:"name"`
}

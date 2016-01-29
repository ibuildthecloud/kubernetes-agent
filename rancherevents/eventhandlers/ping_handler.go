package eventhandlers

import (
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	util "github.com/rancher/kubernetes-agent/rancherevents/util"
)

type PingHandler struct {
}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (h *PingHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	reply := util.NewReply(event)
	if reply.Name == "" {
		return nil
	}
	err := util.PublishReply(reply, cli)
	if err != nil {
		return err
	}
	return nil
}

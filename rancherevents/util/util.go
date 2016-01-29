package eventhandlers

import (
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func NewReply(event *revents.Event) *client.Publish {
	return &client.Publish{
		Name:        event.ReplyTo,
		PreviousIds: []string{event.Id},
	}
}

func PublishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}

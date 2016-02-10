package eventhandlers

import (
	"encoding/json"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	types "github.com/rancher/kubernetes-agent/rancherevents/types"
	"github.com/rancher/kubernetes-model/model"
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

func ConvertRancherToKubernetesService(service *types.Service) (model.Service, error) {
	var kService model.Service
	m, _ := json.Marshal(service.Data.Fields.Template)
	if err := json.Unmarshal(m, &kService); err != nil {
		return kService, err
	}
	if kService.Metadata == nil {
		kService.Metadata = &model.ObjectMeta{
			Name: service.Name,
		}
	}

	return kService, nil
}

func ConvertRancherToKubernetesReplicationController(service *types.Service) (model.ReplicationController, error) {
	var rc model.ReplicationController
	m, _ := json.Marshal(service.Data.Fields.Template)
	if err := json.Unmarshal(m, &rc); err != nil {
		return rc, err
	}

	if rc.Metadata == nil {
		rc.Metadata = &model.ObjectMeta{
			Name: service.Name,
		}
	}

	return rc, nil
}

func GetRancherService(event *revents.Event, cli *client.RancherClient) (types.Service, error) {
	var service types.Service

	data := event.Data
	if svc, ok := data["service"]; ok {
		if svcMap, ok := svc.(map[string]interface{}); ok {
			m, _ := json.Marshal(svcMap)
			if err := json.Unmarshal(m, &service); err != nil {
				return service, err
			}
		}
	}

	return service, nil
}

func GetNewRancherResourceVersion(service *types.Service, metadata *model.ObjectMeta) string {
	newVersion := service.Data.Fields.ResourceVersion
	var existingVersion interface{}
	if metadata.Labels != nil {
		existingVersion = metadata.Labels["io.rancher.resourceversion"]
	}
	if newVersion == existingVersion {
		return ""
	}
	return service.Data.Fields.ResourceVersion
}

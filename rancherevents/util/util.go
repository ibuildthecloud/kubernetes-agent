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

func CreateAndPublishReply(event *revents.Event, cli *client.RancherClient) error {
	reply := NewReply(event)
	if reply.Name == "" {
		return nil
	}
	err := PublishReply(reply, cli)
	if err != nil {
		return err
	}
	return nil
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

	if kService.Metadata.Labels == nil {
		kService.Metadata.Labels = make(map[string]interface{})
	}

	kService.Metadata.Labels["io.rancher.uuid"] = service.UUID

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
	if rc.Metadata.Labels == nil {
		rc.Metadata.Labels = make(map[string]interface{})
	}

	rc.Metadata.Labels["io.rancher.uuid"] = service.UUID

	return rc, nil
}

func GetRancherService(event *revents.Event) (types.Service, error) {
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

func IsNoOp(event *revents.Event) bool {
	data := event.Data
	if process, ok := data["processData"]; ok {
		if svcMap, ok := process.(map[string]interface{}); ok {
			if val, ok := svcMap["containerNoOpEvent"]; ok {
				if boolVal, ok := val.(bool); ok {
					return boolVal
				}
			}
		}
	}
	return false
}

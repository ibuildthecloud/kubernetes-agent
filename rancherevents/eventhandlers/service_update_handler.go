package eventhandlers

import (
	log "github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/kubernetes-agent/kubernetesclient"
	types "github.com/rancher/kubernetes-agent/rancherevents/types"
	util "github.com/rancher/kubernetes-agent/rancherevents/util"
)

type ServiceUpdateHandler struct {
	kClient *kubernetesclient.Client
}

func NewServiceUpdateHandler(kClient *kubernetesclient.Client) *ServiceUpdateHandler {
	return &ServiceUpdateHandler{
		kClient: kClient,
	}
}

func (h *ServiceUpdateHandler) updateKubernetesService(service *types.Service) error {
	log.Infof("updating service %v", service)
	kService, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := kService.Metadata.Name
	_, err = h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err == nil {
		if kService.Metadata.Labels == nil {
			kService.Metadata.Labels = make(map[string]interface{})
		}

		kService.Metadata.Labels["io.rancher.uuid"] = service.UUID
		_, err = h.kClient.Service.ReplaceService(service.Stack.Name, &kService)
		if err != nil {
			return err
		}
		log.Infof("Updated kubernetesService %s", svcName)
	}

	return nil
}

func (h *ServiceUpdateHandler) updateKubernetesReplicationController(service *types.Service) error {
	log.Infof("updating rc %v", service)

	rc, err := util.ConvertRancherToKubernetesReplicationController(service)
	if err != nil {
		return err
	}

	svcName := rc.Metadata.Name
	_, err = h.kClient.ReplicationController.ByName(service.Stack.Name, svcName)
	if err == nil {
		if rc.Metadata.Labels == nil {
			rc.Metadata.Labels = make(map[string]interface{})
		}

		rc.Metadata.Labels["io.rancher.uuid"] = service.UUID
		_, err = h.kClient.ReplicationController.ReplaceReplicationController(service.Stack.Name, &rc)
		if err != nil {
			return err
		}
		log.Infof("Updated kubernetesReplicationController %s", svcName)
	}
	return nil
}

func (svch *ServiceUpdateHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	log.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)

	service, err := util.GetRancherService(event, cli)
	if err != nil {
		return err
	}

	if service.Kind == "kubernetesService" {
		if err = svch.updateKubernetesService(&service); err != nil {
			return err
		}
	} else if service.Kind == "kubernetesReplicationController" {
		if err = svch.updateKubernetesReplicationController(&service); err != nil {
			return err
		}
	}

	reply := util.NewReply(event)
	if reply.Name == "" {
		return nil
	}
	err = util.PublishReply(reply, cli)
	if err != nil {
		return err
	}
	return nil
}

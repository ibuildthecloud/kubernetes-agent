package eventhandlers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/kubernetes-agent/kubernetesclient"
	types "github.com/rancher/kubernetes-agent/rancherevents/types"
	util "github.com/rancher/kubernetes-agent/rancherevents/util"
)

type ServiceRemoveHandler struct {
	kClient *kubernetesclient.Client
}

func NewServiceRemoveHandler(kClient *kubernetesclient.Client) *ServiceRemoveHandler {
	return &ServiceRemoveHandler{
		kClient: kClient,
	}
}

func (h *ServiceRemoveHandler) removeKubernetesService(service *types.Service) error {

	kService, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := kService.Metadata.Name
	log.Infof("Removing kubernetesService %s", svcName)

	_, err = h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err == nil {
		status, err := h.kClient.Service.DeleteService(service.Stack.Name, svcName)
		if err != nil {
			return err
		}
		if status.Code != 200 {
			return fmt.Errorf("Failed to remove kubernetesService %s; response code %v", svcName, status.Code)
		}
		log.Infof("Removed kubernetesService %s", svcName)
	}

	return nil
}

func (h *ServiceRemoveHandler) removeKubernetesReplicationController(service *types.Service) error {
	_, err := h.kClient.ReplicationController.ByName(service.Stack.Name, service.Name)

	if err == nil {
		status, err := h.kClient.ReplicationController.DeleteReplicationController(service.Stack.Name, service.Name)
		if err != nil {
			return err
		}
		if status.Code != 200 {
			return fmt.Errorf("Failed to remove kubernetesReplicationController %s; response code %v", service.Name, status.Code)
		}
		log.Infof("Removed kubernetesReplicationController %s", service.Name)
	}
	return nil
}

func (svch *ServiceRemoveHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	log.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)

	service, err := util.GetRancherService(event, cli)
	if err != nil {
		return err
	}

	if service.Kind == "kubernetesService" {
		if err = svch.removeKubernetesService(&service); err != nil {
			return err
		}
	} else if service.Kind == "kubernetesReplicationController" {
		if err = svch.removeKubernetesReplicationController(&service); err != nil {
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

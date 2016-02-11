package eventhandlers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/kubernetes-agent/kubernetesclient"
	types "github.com/rancher/kubernetes-agent/rancherevents/types"
	util "github.com/rancher/kubernetes-agent/rancherevents/util"
	"github.com/rancher/kubernetes-model/model"
	"strings"
)

type ServiceHandler struct {
	kClient *kubernetesclient.Client
}

func NewServiceHandler(kClient *kubernetesclient.Client) *ServiceHandler {
	return &ServiceHandler{
		kClient: kClient,
	}
}

func (h *ServiceHandler) updateKubernetesService(service *types.Service) error {
	toUpdate, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := toUpdate.Metadata.Name
	existing, err := h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err == nil {
		toUpdate.Metadata.ResourceVersion = existing.Metadata.ResourceVersion

		if _, err = h.kClient.Service.ReplaceService(service.Stack.Name, &toUpdate); err != nil {
			return err
		}
	}

	return nil
}

func (h *ServiceHandler) updateKubernetesReplicationController(service *types.Service) error {
	toUpdate, err := util.ConvertRancherToKubernetesReplicationController(service)
	if err != nil {
		return err
	}

	svcName := toUpdate.Metadata.Name
	existing, err := h.kClient.ReplicationController.ByName(service.Stack.Name, svcName)
	toUpdate.Metadata.ResourceVersion = existing.Metadata.ResourceVersion
	if err == nil {
		if _, err = h.kClient.ReplicationController.ReplaceReplicationController(service.Stack.Name, &toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func (h *ServiceHandler) removeKubernetesService(service *types.Service) error {

	kService, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := kService.Metadata.Name

	_, err = h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err == nil {
		status, err := h.kClient.Service.DeleteService(service.Stack.Name, svcName)
		if err != nil {
			return err
		}
		if status.Code != 200 {
			return fmt.Errorf("Failed to remove kubernetesService %s; response code %v", svcName, status.Code)
		}
	}

	return nil
}

func (h *ServiceHandler) removeKubernetesReplicationController(service *types.Service) error {
	_, err := h.kClient.ReplicationController.ByName(service.Stack.Name, service.Name)

	if err == nil {
		status, err := h.kClient.ReplicationController.DeleteReplicationController(service.Stack.Name, service.Name)
		if err != nil {
			return err
		}
		if status.Code != 200 {
			return fmt.Errorf("Failed to remove kubernetesReplicationController %s; response code %v", service.Name, status.Code)
		}
	}
	return nil
}

func (h *ServiceHandler) createKubernetesNameSpace(stack types.Stack) error {
	_, err := h.kClient.Namespace.ByName(stack.Name)
	if err != nil {
		namespace := &model.Namespace{
			Metadata: &model.ObjectMeta{Name: strings.ToLower(stack.Name), Labels: map[string]interface{}{
				"io.rancher.uuid": stack.UUID,
			}},
		}
		if _, err := h.kClient.Namespace.CreateNamespace(namespace); err != nil {
			return err
		}
		log.Infof("Created namespace %s", stack.Name)
	}
	return nil
}

func (h *ServiceHandler) createKubernetesService(service *types.Service) error {
	kService, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := kService.Metadata.Name
	_, err = h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err != nil {
		if _, err = h.kClient.Service.CreateService(service.Stack.Name, &kService); err != nil {
			return err
		}
	}

	return nil
}

func (h *ServiceHandler) createKubernetesReplicationController(service *types.Service) error {
	rc, err := util.ConvertRancherToKubernetesReplicationController(service)
	if err != nil {
		return err
	}

	svcName := rc.Metadata.Name
	_, err = h.kClient.ReplicationController.ByName(service.Stack.Name, svcName)
	if err != nil {
		if _, err = h.kClient.ReplicationController.CreateReplicationController(service.Stack.Name, &rc); err != nil {
			return err
		}
	}
	return nil
}

func (h *ServiceHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	log.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)
	if util.IsNoOp(event) {
		log.Infof("Not processing noop event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)
	} else {
		service, err := util.GetRancherService(event)
		if err != nil {
			return err
		}

		if event.Name == "service.create" {
			if err := h.createKubernetesNameSpace(service.Stack); err != nil {
				return err
			}
		}

		if service.Kind == "kubernetesService" {
			if event.Name == "service.create" {
				if err = h.createKubernetesService(&service); err != nil {
					return err
				}
			} else if event.Name == "service.update" {
				if err = h.updateKubernetesService(&service); err != nil {
					return err
				}
			} else if event.Name == "service.remove" {
				if err = h.removeKubernetesService(&service); err != nil {
					return err
				}
			}

		} else if service.Kind == "kubernetesReplicationController" {
			if event.Name == "service.create" {
				if err = h.createKubernetesReplicationController(&service); err != nil {
					return err
				}
			} else if event.Name == "service.update" {
				if err = h.updateKubernetesReplicationController(&service); err != nil {
					return err
				}
			} else if event.Name == "service.remove" {
				if err = h.removeKubernetesReplicationController(&service); err != nil {
					return err
				}
			}
		}
	}

	if err := util.CreateAndPublishReply(event, cli); err != nil {
		return err
	}
	return nil
}

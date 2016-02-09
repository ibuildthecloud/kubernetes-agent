package eventhandlers

import (
	log "github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/kubernetes-agent/kubernetesclient"
	types "github.com/rancher/kubernetes-agent/rancherevents/types"
	util "github.com/rancher/kubernetes-agent/rancherevents/util"
	"github.com/rancher/kubernetes-model/model"
	"strings"
)

type ServiceCreateHandler struct {
	kClient *kubernetesclient.Client
}

func NewServiceCreateHandler(kClient *kubernetesclient.Client) *ServiceCreateHandler {
	return &ServiceCreateHandler{
		kClient: kClient,
	}
}

func (h *ServiceCreateHandler) createKubernetesNameSpace(stack types.Stack) error {
	_, err := h.kClient.Namespace.ByName(stack.Name)
	if err != nil {
		namespace := &model.Namespace{
			Metadata: &model.ObjectMeta{Name: strings.ToLower(stack.Name), Labels: map[string]interface{}{
				"io.rancher.uuid": stack.UUID,
			}},
		}
		_, err := h.kClient.Namespace.CreateNamespace(namespace)
		if err != nil {
			return err
		}
		log.Infof("Created namespace %s", stack.Name)
	}
	return nil
}

func (h *ServiceCreateHandler) createKubernetesService(service *types.Service) error {
	kService, err := util.ConvertRancherToKubernetesService(service)
	if err != nil {
		return err
	}

	svcName := kService.Metadata.Name
	_, err = h.kClient.Service.ByName(service.Stack.Name, svcName)
	if err != nil {
		if kService.Metadata.Labels == nil {
			kService.Metadata.Labels = make(map[string]interface{})
		}

		kService.Metadata.Labels["io.rancher.uuid"] = service.UUID
		_, err = h.kClient.Service.CreateService(service.Stack.Name, &kService)
		if err != nil {
			return err
		}
		log.Infof("Created kubernetesService %s", svcName)
	}

	return nil
}

func (h *ServiceCreateHandler) createKubernetesReplicationController(service *types.Service) error {
	rc, err := util.ConvertRancherToKubernetesReplicationController(service)
	if err != nil {
		return err
	}

	svcName := rc.Metadata.Name
	_, err = h.kClient.ReplicationController.ByName(service.Stack.Name, svcName)
	if err != nil {
		if rc.Metadata.Labels == nil {
			rc.Metadata.Labels = make(map[string]interface{})
		}

		rc.Metadata.Labels["io.rancher.uuid"] = service.UUID
		_, err = h.kClient.ReplicationController.CreateReplicationController(service.Stack.Name, &rc)
		if err != nil {
			return err
		}
		log.Infof("Created kubernetesReplicationController %s", svcName)
	}
	return nil
}

func (svch *ServiceCreateHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	log.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)

	service, err := util.GetRancherService(event, cli)
	if err != nil {
		return err
	}

	if err := svch.createKubernetesNameSpace(service.Stack); err != nil {
		return err
	}

	if service.Kind == "kubernetesService" {
		if err = svch.createKubernetesService(&service); err != nil {
			return err
		}
	} else if service.Kind == "kubernetesReplicationController" {
		if err = svch.createKubernetesReplicationController(&service); err != nil {
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

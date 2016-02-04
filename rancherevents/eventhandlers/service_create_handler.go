package eventhandlers

import (
	"encoding/json"
	"errors"
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

func (h *ServiceCreateHandler) getRancherService(event *revents.Event, cli *client.RancherClient) (types.Service, error) {
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

func getServicePorts(service *types.Service) []model.ServicePort {
	ports := make([]model.ServicePort, 0)

	for _, port := range service.Data.Fields.Ports {
		port := model.ServicePort{
			Protocol:   strings.ToUpper(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort,
			NodePort:   port.NodePort,
			Name:       port.Name,
		}
		ports = append(ports, port)
	}
	return ports
}

func getServiceSelector(service *types.Service) map[string]interface{} {
	kSelector := map[string]interface{}{}
	if service.SelectorContainer == "" {
		return kSelector
	}
	selector := strings.TrimSpace(service.SelectorContainer)
	splitted := strings.SplitN(selector, "=", 2)
	// only equality based selectors are supported
	if splitted[1] == "" || strings.HasSuffix(splitted[0], "!") {
		return kSelector
	}
	kSelector[strings.TrimSpace(splitted[0])] = strings.TrimSpace(splitted[1])

	return kSelector
}

func (h *ServiceCreateHandler) createKubernetesService(service *types.Service) error {
	_, err := h.kClient.Service.ByName(service.Stack.Name, service.Name)

	if err != nil {
		spec := &model.ServiceSpec{
			Type:            service.Data.Fields.Type,
			SessionAffinity: service.Data.Fields.SessionAffinity,
			ExternalIPs:     service.Data.Fields.ExternalIPs,
			ClusterIP:       service.Data.Fields.ClusterIP,
			Selector:        getServiceSelector(service),
			Ports:           getServicePorts(service),
		}

		labels := map[string]interface{}{
			"io.rancher.uuid": service.UUID}

		sLabels := service.Data.Fields.Labels
		if sLabels != nil {
			for k, v := range sLabels {
				labels[k] = v
			}
		}

		kService := &model.Service{
			Metadata: &model.ObjectMeta{Name: service.Name, Labels: labels},
			Spec:     spec,
		}

		_, err := h.kClient.Service.CreateService(service.Stack.Name, kService)
		if err != nil {
			return err
		}
		log.Infof("Created kubernetesService %s", service.Name)
	}
	return nil
}

func getPodTemplate(service *types.Service) (*model.PodTemplateSpec, error) {
	containers := make([]model.Container, 0)
	splitted := strings.SplitN(service.Data.Fields.LaunchConfig.Image, "docker:", 2)
	if len(splitted) < 2 {
		return nil, errors.New("Image is missing on a service")
	}
	container := model.Container{
		Name:  service.Data.Fields.LaunchConfig.Name,
		Image: splitted[1],
	}

	containers = append(containers, container)

	podSpec := &model.PodSpec{
		ActiveDeadlineSeconds:         service.Data.Fields.ActiveDeadlineSeconds,
		DnsPolicy:                     service.Data.Fields.DnsPolicy,
		HostIPC:                       service.Data.Fields.HostIPC,
		HostNetwork:                   service.Data.Fields.HostNetwork,
		HostPID:                       service.Data.Fields.HostPID,
		NodeName:                      service.Data.Fields.NodeName,
		NodeSelector:                  nil,
		ServiceAccountName:            service.Data.Fields.ServiceAccountName,
		TerminationGracePeriodSeconds: service.Data.Fields.TerminationGracePeriodSeconds,
		Containers:                    containers,
	}

	template := &model.PodTemplateSpec{
		Metadata: &model.ObjectMeta{Name: service.Name, Labels: service.Data.Fields.LaunchConfig.Labels},
		Spec:     podSpec,
	}
	return template, nil
}

func (h *ServiceCreateHandler) createKubernetesReplicationController(service *types.Service) error {
	_, err := h.kClient.ReplicationController.ByName(service.Stack.Name, service.Name)

	if err != nil {
		labels := map[string]interface{}{
			"io.rancher.uuid": service.UUID}

		sLabels := service.Data.Fields.Labels
		if sLabels != nil {
			for k, v := range sLabels {
				labels[k] = v
			}
		}
		podTemplateSpec, err := getPodTemplate(service)
		if err != nil {
			return err
		}
		svcSpec := &model.ReplicationControllerSpec{
			Selector: getServiceSelector(service),
			Replicas: service.Scale,
			Template: podTemplateSpec,
		}

		kRC := &model.ReplicationController{
			Metadata: &model.ObjectMeta{Name: service.Name, Labels: labels},
			Spec:     svcSpec,
		}
		log.Infof("labels are %v", labels)
		log.Infof("selector is %v", svcSpec.Selector)
		_, err = h.kClient.ReplicationController.CreateReplicationController(service.Stack.Name, kRC)
		if err != nil {
			return err
		}
		log.Infof("Created kubernetesReplicationController %s", service.Name)
	}
	return nil
}

func (svch *ServiceCreateHandler) Handler(event *revents.Event, cli *client.RancherClient) error {
	log.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.Id, event.ResourceId)

	service, err := svch.getRancherService(event, cli)
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

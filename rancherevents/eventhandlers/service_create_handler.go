package eventhandlers

import (
	"encoding/json"
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
	if service.Selector == "" {
		return kSelector
	}
	selector := strings.TrimSpace(service.Selector)
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

		lc := service.Data.Fields.LaunchConfig
		if &lc != nil {
			if lc.Labels != nil {
				for k, v := range lc.Labels {
					labels[k] = v
				}
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

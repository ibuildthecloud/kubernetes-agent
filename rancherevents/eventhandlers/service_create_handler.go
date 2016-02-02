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
	"strconv"
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
			Metadata: &model.ObjectMeta{Name: stack.Name, Labels: map[string]interface{}{
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

func getServiceLaunchConfig(service *types.Service) types.LaunchConfig {
	var launchConfig types.LaunchConfig
	if f, ok := service.Data["fields"]; ok {
		if fMap, ok := f.(map[string]interface{}); ok {
			if lc, ok := fMap["launchConfig"]; ok {
				m, _ := json.Marshal(lc)
				json.Unmarshal(m, &launchConfig)
			}
		}
	}

	return launchConfig
}

func getServicePorts(service *types.Service) []model.ServicePort {
	launchConfig := getServiceLaunchConfig(service)
	ports := make([]model.ServicePort, 0)

	for _, port := range launchConfig.Ports {
		var proto string
		var sourcePort string
		var targetPort string
		splitted := strings.SplitN(port, "/", 2)
		if splitted[1] != "" {
			proto = splitted[1]
		} else {
			proto = "TCP"
		}
		splitted = strings.SplitN(splitted[0], ":", 2)
		sourcePort = splitted[0]
		if splitted[1] == "" {
			targetPort = sourcePort
		} else {
			targetPort = splitted[1]
		}

		sp, _ := strconv.ParseInt(sourcePort, 10, 32)
		tp, _ := strconv.ParseInt(targetPort, 10, 32)
		spt := int32(sp)
		tpt := int32(tp)
		ptcl := strings.ToUpper(proto)
		port := model.ServicePort{
			Protocol:   ptcl,
			Port:       spt,
			TargetPort: tpt,
			Name:       sourcePort,
		}
		ports = append(ports, port)
	}
	return ports
}

func getServiceSelector(service *types.Service) map[string]interface{} {
	selector := map[string]interface{}{}
	//FIXME - populate selector
	return selector
}

func (h *ServiceCreateHandler) createKubernetesService(service *types.Service) error {
	_, err := h.kClient.Service.ByName(service.Stack.Name, service.Name)

	if err != nil {
		spec := &model.ServiceSpec{
			Type:            service.Type,
			SessionAffinity: service.SessionAffinity,
			ExternalIPs:     service.ExternalIPs,
			LoadBalancerIP:  service.LoadBalancerIP,
			ClusterIP:       service.ClusterIP,
			Selector:        getServiceSelector(service),
			Ports:           getServicePorts(service),
		}
		kService := &model.Service{
			Metadata: &model.ObjectMeta{Name: service.Name, Labels: map[string]interface{}{
				"io.rancher.uuid": service.UUID}},
			Spec: spec,
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

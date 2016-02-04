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
	return parseSelector(service.SelectorContainer)
}

func parseSelector(selector string) map[string]interface{} {
	kSelector := map[string]interface{}{}
	if selector == "" {
		return kSelector
	}
	selector = strings.TrimSpace(selector)
	splitted := strings.SplitN(selector, "=", 2)
	// only equality based selectors are supported
	if len(splitted) < 2 || strings.HasSuffix(splitted[0], "!") {
		return kSelector
	}
	kSelector[strings.TrimSpace(splitted[0])] = strings.TrimSpace(splitted[1])

	return kSelector
}

func (h *ServiceCreateHandler) createKubernetesService(service *types.Service) error {
	_, err := h.kClient.ReplicationController.ByName(service.Stack.Name, service.Name)

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

func getValueFromLabels(name string, lc *client.LaunchConfig) string {
	sLabels := lc.Labels
	if sLabels != nil {
		if val, ok := sLabels[name]; ok {
			if valStr, ok := val.(string); ok {
				return valStr
			}
		}
	}
	return ""
}

func getCommandAndArgs(lc *client.LaunchConfig) ([]string, []string) {
	command := make([]string, 0)
	args := make([]string, 0)

	fullCmd := lc.Command
	if len(fullCmd) == 0 {
		copy(command[:], lc.EntryPoint)
	} else {
		command = append(command, fullCmd[0])
		for index, arg := range fullCmd {
			if index == 0 {
				continue
			}
			args = append(args, arg)
		}
	}
	return command, args
}

func getEnvVars(lc *client.LaunchConfig) []model.EnvVar {
	envVars := make([]model.EnvVar, 0)
	for key, value := range lc.Environment {
		if valueStr, ok := value.(string); ok {
			envVar := model.EnvVar{
				Name:  key,
				Value: valueStr,
			}
			envVars = append(envVars, envVar)
		}
	}
	return envVars
}

func getContainerPorts(lc *client.LaunchConfig) []model.ContainerPort {
	ports := make([]model.ContainerPort, 0)
	for _, port := range lc.Ports {
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
		port := model.ContainerPort{
			Protocol:      ptcl,
			HostPort:      spt,
			ContainerPort: tpt,
			Name:          sourcePort,
		}
		ports = append(ports, port)
	}
	return ports
}

func getLaunchCofigs(service *types.Service) []client.LaunchConfig {
	lcs := make([]client.LaunchConfig, 0)
	copy(lcs[:], service.Data.Fields.SecondaryLaunchConfigs)
	plc := service.Data.Fields.LaunchConfig
	lcs = append(lcs, plc)
	return lcs
}

func getSecurityContext(lc *client.LaunchConfig) *model.SecurityContext {
	add := make([]model.Capability, 0)
	drop := make([]model.Capability, 0)
	for _, rCap := range lc.CapAdd {
		add = append(add, model.Capability(rCap))
	}

	for _, rCap := range lc.CapDrop {
		drop = append(drop, model.Capability(rCap))
	}

	caps := &model.Capabilities{
		Add:  add,
		Drop: drop,
	}

	ctx := &model.SecurityContext{
		Capabilities: caps,
		Privileged:   lc.Privileged,
		//RunAsUser:      nil, //done
		//RunAsNonRoot:   nil, //done
		//SeLinuxOptions: nil,
	}
	return ctx
}

func getContainer(lc *client.LaunchConfig, mounts []model.VolumeMount) (*model.Container, error) {
	splitted := strings.SplitN(lc.ImageUuid, "docker:", 2)
	if len(splitted) < 2 {
		return nil, errors.New("Image is missing on a service")
	}

	imagePullPolicy := getValueFromLabels("io.rancher.container.pull_image", lc)
	if imagePullPolicy == "always" {
		imagePullPolicy = "Always"
	} else {
		imagePullPolicy = ""
	}

	command, args := getCommandAndArgs(lc)
	container := model.Container{
		//FIXME: read name from lc
		Name:            "foo",
		Image:           splitted[1],
		Stdin:           lc.StdinOpen,
		Tty:             lc.Tty,
		Command:         command,
		ImagePullPolicy: imagePullPolicy,
		Args:            args,
		Env:             getEnvVars(lc),
		WorkingDir:      lc.WorkingDir,
		Ports:           getContainerPorts(lc),
		VolumeMounts:    mounts,
	}

	/*Lifecycle:                      nil, //done
				  LivenessProbe:          nil,//done
				  ReadinessProbe:         nil,//done
				  TerminationMessagePath: nil,//done
				  Resources:              nil,//done
		SecurityContext: nil //done
	}*/
	return &container, nil
}

func getContainersAndVolumes(service *types.Service) ([]model.Container, []model.Volume, error) {
	mounts, volumes, err := getVolumes(service)
	if err != nil {
		return nil, nil, err
	}
	containers := make([]model.Container, 0)
	for _, lc := range getLaunchCofigs(service) {
		//fix me - use launch config name
		if container, err := getContainer(&lc, mounts["foo"]); err != nil {
			return nil, nil, err
		} else {
			containers = append(containers, *container)
		}
	}

	return containers, volumes, nil
}

func getHostNetwork(service *types.Service) bool {
	return service.Data.Fields.LaunchConfig.NetworkMode == "host"
}

func getHostPID(service *types.Service) bool {
	return service.Data.Fields.LaunchConfig.PidMode == "host"
}

func getImagePullSecrets(service *types.Service) []model.LocalObjectReference {
	secrets := make([]model.LocalObjectReference, 0)
	for _, name := range service.Data.Fields.ImagePullSecrets {
		secret := model.LocalObjectReference{
			Name: name,
		}
		secrets = append(secrets, secret)
	}
	return secrets
}

func getVolumes(service *types.Service) (map[string][]model.VolumeMount, []model.Volume, error) {
	lcNameToMount := make(map[string][]model.VolumeMount)
	vs := make([]model.Volume, 0)
	lcs := getLaunchCofigs(service)
	for _, lc := range lcs {
		i := 0
		ms := make([]model.VolumeMount, 0)
		//FIXME read from the real config name
		lcName := "foo"
		for _, vol := range lc.DataVolumes {
			log.Infof("volume is %v", vol)
			splitted := strings.Split(vol, ":")
			if len(splitted) < 2 {
				return nil, nil, errors.New("DataVolume mount should have host and container path")
			}
			//fix me - read from config name
			name := lcName + strconv.Itoa(i)
			readOnly := false
			if len(splitted) == 3 && splitted[3] == "ro" {
				readOnly = true
			}

			hp := &model.HostPathVolumeSource{
				Path: splitted[0],
			}
			v := model.Volume{
				Name:     name,
				HostPath: hp,
			}
			vs = append(vs, v)

			m := model.VolumeMount{
				Name:      name,
				MountPath: splitted[1],
				ReadOnly:  readOnly,
			}

			ms = append(ms, m)
			i++
		}
		lcNameToMount[lcName] = ms
		i = 0
	}
	return lcNameToMount, vs, nil
}

func getPodTemplate(service *types.Service) (*model.PodTemplateSpec, error) {
	containers, volumes, err := getContainersAndVolumes(service)
	if err != nil {
		return nil, err
	}

	svc := &service.Data.Fields
	podSpec := &model.PodSpec{
		ActiveDeadlineSeconds:         svc.ActiveDeadlineSeconds,
		DnsPolicy:                     svc.DnsPolicy,
		HostIPC:                       svc.HostIPC,
		HostNetwork:                   getHostNetwork(service),
		HostPID:                       getHostPID(service),
		NodeName:                      svc.NodeName,
		ServiceAccountName:            svc.ServiceAccountName,
		TerminationGracePeriodSeconds: svc.TerminationGracePeriodSeconds,
		Containers:                    containers,
		NodeSelector:                  parseSelector(svc.NodeSelector),
		ImagePullSecrets:              getImagePullSecrets(service),
		Volumes:                       volumes,
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

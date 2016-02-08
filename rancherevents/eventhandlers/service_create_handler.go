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

func getServiceSelector(service *types.Service) map[string]interface{} {
	return parseSelector(service.SelectorContainer)
}

func parseSelector(selector string) map[string]interface{} {
	kSelector := map[string]interface{}{}
	if selector == "" {
		return kSelector
	}
	selectors := strings.Split(selector, ",")
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		splitted := strings.SplitN(selector, "=", 2)
		// only equality based selectors are supported
		if len(splitted) < 2 || strings.HasSuffix(splitted[0], "!") {
			continue
		}
		kSelector[strings.TrimSpace(splitted[0])] = strings.TrimSpace(splitted[1])
	}

	return kSelector
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

func getValueFromLabels(name string, lc *client.KubernetesLaunchConfig) string {
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

func getCommandAndArgs(lc *client.KubernetesLaunchConfig) ([]string, []string) {
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

func getEnvVars(lc *client.KubernetesLaunchConfig) []model.EnvVar {
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

func getContainerPorts(lc *client.KubernetesLaunchConfig) []model.ContainerPort {
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

func getLaunchCofigs(service *types.Service) []client.KubernetesLaunchConfig {
	lcs := make([]client.KubernetesLaunchConfig, 0)
	copy(lcs[:], service.Data.Fields.SecondaryLaunchConfigs)
	plc := service.Data.Fields.LaunchConfig
	lcs = append(lcs, plc)
	return lcs
}

func getSecurityContext(lc *client.KubernetesLaunchConfig) *model.SecurityContext {
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
		RunAsUser:    lc.RunAsUser,
		RunAsNonRoot: lc.RunAsNonRoot,
		//SeLinuxOptions: nil,
	}
	return ctx
}

func getLifecycle(lc *client.KubernetesLaunchConfig) (*model.Lifecycle, error) {
	var lifecycle model.Lifecycle

	m, _ := json.Marshal(lc.Lifecycle)
	if err := json.Unmarshal(m, &lifecycle); err != nil {
		return nil, err
	}

	if lifecycle.PostStart != nil {
		i := 0
		if len(lifecycle.PostStart.Exec.Command) == 0 {
			lifecycle.PostStart.Exec = nil
			i++
		}

		if lifecycle.PostStart.HttpGet.Port == 0 {
			lifecycle.PostStart.HttpGet = nil
			i++
		}

		if lifecycle.PostStart.TcpSocket.Port == "" {
			lifecycle.PostStart.TcpSocket = nil
			i++
		}
		if i == 3 {
			lifecycle.PostStart = nil
		}
	}

	if lifecycle.PreStop != nil {
		i := 0
		if len(lifecycle.PreStop.Exec.Command) == 0 {
			lifecycle.PreStop.Exec = nil
			i++
		}

		if lifecycle.PreStop.HttpGet.Port == 0 {
			lifecycle.PreStop.HttpGet = nil
			i++
		}

		if lifecycle.PreStop.TcpSocket.Port == "" {
			lifecycle.PreStop.TcpSocket = nil
			i++
		}
		if i == 3 {
			lifecycle.PreStop = nil
		}
	}
	if lifecycle.PreStop == nil && lifecycle.PostStart == nil {
		return nil, nil
	}

	return &lifecycle, nil
}

func getRequirements(lc *client.KubernetesLaunchConfig) (*model.ResourceRequirements, error) {
	var req model.ResourceRequirements
	m, _ := json.Marshal(lc.Resources)
	if err := json.Unmarshal(m, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func getProbe(rancherProbe *client.KubernetesProbe) (*model.Probe, error) {
	var probe model.Probe

	m, _ := json.Marshal(rancherProbe)
	if err := json.Unmarshal(m, &probe); err != nil {
		return nil, err
	}

	i := 0
	if len(probe.Exec.Command) == 0 {
		probe.Exec = nil
		i++
	}

	if probe.HttpGet.Port == 0 {
		probe.HttpGet = nil
		i++
	}

	if probe.TcpSocket.Port == "" {
		probe.TcpSocket = nil
		i++
	}

	if i == 3 {
		return nil, nil
	}

	return &probe, nil
}

func getContainer(lc *client.KubernetesLaunchConfig, mounts []model.VolumeMount) (*model.Container, error) {
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
	lifecycle, err := getLifecycle(lc)
	if err != nil {
		return nil, err
	}
	req, err := getRequirements(lc)
	if err != nil {
		return nil, err
	}

	lProbe, err := getProbe(&lc.LivenessProbe)
	if err != nil {
		return nil, err
	}

	rProbe, err := getProbe(&lc.ReadinessProbe)
	if err != nil {
		return nil, err
	}

	container := model.Container{
		Name:                   lc.Name,
		Image:                  splitted[1],
		Stdin:                  lc.StdinOpen,
		Tty:                    lc.Tty,
		Command:                command,
		ImagePullPolicy:        imagePullPolicy,
		Args:                   args,
		Env:                    getEnvVars(lc),
		WorkingDir:             lc.WorkingDir,
		Ports:                  getContainerPorts(lc),
		VolumeMounts:           mounts,
		Resources:              req,
		TerminationMessagePath: lc.TerminationMessagePath,
		SecurityContext:        getSecurityContext(lc),
		Lifecycle:              lifecycle,
		LivenessProbe:          lProbe,
		ReadinessProbe:         rProbe,
	}

	return &container, nil
}

func getContainersAndVolumes(service *types.Service) ([]model.Container, []model.Volume, error) {
	mounts, volumes, err := getVolumes(service)
	if err != nil {
		return nil, nil, err
	}
	containers := make([]model.Container, 0)
	for _, lc := range getLaunchCofigs(service) {
		if container, err := getContainer(&lc, mounts[lc.Name]); err != nil {
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
		lcName := lc.Name
		for _, vol := range lc.DataVolumes {
			log.Infof("volume is %v", vol)
			splitted := strings.Split(vol, ":")
			if len(splitted) < 2 {
				return nil, nil, errors.New("DataVolume mount should have host and container path")
			}
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

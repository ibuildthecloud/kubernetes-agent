package rancherevents

import (
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/kubernetes-agent/config"
	"github.com/rancher/kubernetes-agent/kubernetesclient"
	"github.com/rancher/kubernetes-agent/rancherevents/eventhandlers"
)

func ConnectToEventStream(conf config.Config) error {

	kClient := kubernetesclient.NewClient(conf.KubernetesURL, false)

	eventHandlers := map[string]revents.EventHandler{
		"compute.instance.providelabels": eventhandlers.NewProvideLablesHandler(kClient).Handler,
		"config.update":                  eventhandlers.NewPingHandler().Handler,
		"ping":                           eventhandlers.NewPingHandler().Handler,
		"service.create":                 eventhandlers.NewServiceCreateHandler(kClient).Handler,
		"service.activate":               eventhandlers.NewPingHandler().Handler,
		"service.deactivate":             eventhandlers.NewPingHandler().Handler,
		"service.remove":                 eventhandlers.NewServiceRemoveHandler(kClient).Handler,
		//"service.update":                 eventhandlers.NewServiceUpdateHandler(kClient).Handler,
	}

	router, err := revents.NewEventRouter("", 0, conf.CattleURL, conf.CattleAccessKey, conf.CattleSecretKey, nil, eventHandlers, "", conf.WorkerCount)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}

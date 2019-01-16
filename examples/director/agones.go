package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	backend "github.com/GoogleCloudPlatform/open-match/internal/pb"
)

// AgonesAllocator allocates game servers in Agones fleet
type AgonesAllocator struct {
	agonesClient *versioned.Clientset

	namespace    string
	fleetName    string
	generateName string

	logger *logrus.Entry
}

// NewAgonesAllocator creates new AgonesAllocator with in cluster k8s config
func NewAgonesAllocator(namespace, fleetName, generateName string, logger *logrus.Entry) (*AgonesAllocator, error) {
	agonesClient, err := getAgonesClient()
	if err != nil {
		return nil, errors.New("Could not create Agones allocator: " + err.Error())
	}

	a := &AgonesAllocator{
		agonesClient: agonesClient,

		namespace:    namespace,
		fleetName:    fleetName,
		generateName: generateName,

		logger: logger.WithFields(logrus.Fields{
			"source":       "agones",
			"namespace":    namespace,
			"fleetname":    fleetName,
			"generatename": generateName,
		}),
	}
	return a, nil
}

// Set up our client which we will use to call the API
func getAgonesClient() (*versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.New("Could not create in cluster config: " + err.Error())
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.New("Could not create the agones api clientset: " + err.Error())
	}
	return agonesClient, nil
}

// Allocate allocates a game server in a fleet, distributes match object details to it,
// and returns a connection string or error
func (a *AgonesAllocator) Allocate(match *backend.MatchObject) (string, error) {
	fa, err := a.allocateFleet()
	if err != nil {
		return "", err
	}

	// TODO distribute the results to DGS

	dgs := fa.Status.GameServer.Status
	connstring := fmt.Sprintf("%s:%d", dgs.Address, dgs.Ports[0].Port)

	a.logger.WithFields(logrus.Fields{
		"fleetallocation": fa.Name,
		"gameserver":      fa.Status.GameServer.Name,
		"connstring":      connstring,
	}).Info("GameServer allocated")

	return connstring, nil
}

// Move a replica from ready to allocated and return the FleetAllocation
func (a *AgonesAllocator) allocateFleet() (*v1alpha1.FleetAllocation, error) {
	// TODO doest it really make sense to check the number of replicas before trying to allocate one?
	// ----------------------------------------------------
	// Find out how many ready replicas the fleet has - we need at least one
	// readyReplicas := a.checkReadyReplicas()
	// a.logger.WithField("readyReplicas", readyReplicas).Debug("numer of ready replicas")
	// if readyReplicas < 1 {
	// 	return nil, errors.New("Insufficient ready replicas, cannot create fleet allocation")
	// }

	// Define the fleet allocation using the constants set earlier
	faReq := &v1alpha1.FleetAllocation{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: a.generateName, Namespace: a.namespace,
		},
		Spec: v1alpha1.FleetAllocationSpec{FleetName: a.fleetName},
	}

	// Create a new fleet allocation
	fa, err := a.agonesClient.StableV1alpha1().FleetAllocations(a.namespace).Create(faReq)
	if err != nil {
		return nil, errors.New("Failed to create fleet allocation: " + err.Error())
	}
	return fa, nil
}

// Return the number of ready game servers available to this fleet for allocation
func (a *AgonesAllocator) checkReadyReplicas() int32 {
	fleet, err := a.agonesClient.StableV1alpha1().Fleets(a.namespace).Get(a.fleetName, v1.GetOptions{})
	if err != nil {
		a.logger.WithError(err).Error("Get fleet failed")
		return -1
	}
	return fleet.Status.ReadyReplicas
}

// UnAllocate finds and deletes the allocated game server matching the specified connection string
func (a *AgonesAllocator) UnAllocate(connstring string) error {
	var ip, port string
	if parts := strings.Split(connstring, ":"); len(parts) != 2 {
		return errors.New("unable to parse connection string: expecting format \"<IP>:<PORT>\"")
	} else {
		ip, port = parts[0], parts[1]
	}

	gsi := a.agonesClient.StableV1alpha1().GameServers(a.namespace)

	gsl, err := gsi.List(v1.ListOptions{})
	if err != nil {
		return errors.New("failed to get game servers list: " + err.Error())
	}

	var gameServer *v1alpha1.GameServer
	for _, gs := range gsl.Items {
		if gs.Status.State == "Allocated" && gs.Status.Address == ip {
			for _, p := range gs.Status.Ports {
				if strconv.Itoa(int(p.Port)) == port {
					copy := gs
					gameServer = &copy
					break
				}
			}
		}
	}
	if gameServer == nil {
		return errors.New("found no game servers matching the connection string")
	}

	fields := logrus.Fields{
		"connstring": connstring,
		"gameserver": gameServer.Name,
	}

	err = gsi.Delete(gameServer.Name, nil)
	if err != nil {
		msg := "failed to delete game server"
		a.logger.WithFields(fields).WithError(err).Error(msg)
		return errors.New(msg + ": " + err.Error())
	}
	a.logger.WithFields(fields).Info("GameServer deleted")

	return nil
}
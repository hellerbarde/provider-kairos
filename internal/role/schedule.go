package role

import (
	"math/rand"
	"time"

	"github.com/kairos-io/kairos/pkg/config"

	providerConfig "github.com/kairos-io/provider-kairos/internal/provider/config"
	service "github.com/mudler/edgevpn/api/client/service"
)

// scheduleRoles assigns roles to nodes. Meant to be called only by leaders
// TODO: HA-Auto.
func scheduleRoles(nodes []string, c *service.RoleConfig, cc *config.Config, pconfig *providerConfig.Config) error {
	rand.Seed(time.Now().Unix())

	// Assign roles to nodes
	unassignedNodes, currentRoles := getRoles(c.Client, nodes)
	c.Logger.Infof("I'm the leader. My UUID is: %s.\n Current assigned roles: %+v", c.UUID, currentRoles)

	existsMaster := false

	masterRole := "master"
	workerRole := "worker"

	if pconfig.Kairos.Hybrid {
		c.Logger.Info("hybrid p2p with KubeVIP enabled")
	}

	for _, r := range currentRoles {
		if r == masterRole {
			existsMaster = true
		}
	}
	c.Logger.Infof("Master already present: %t", existsMaster)
	c.Logger.Infof("Unassigned nodes: %+v", unassignedNodes)

	if !existsMaster && len(unassignedNodes) > 0 {
		var selected string
		toSelect := unassignedNodes

		// Avoid to schedule to ourselves if we have a static role
		if pconfig.Kairos.Role != "" {
			toSelect = []string{}
			for _, u := range unassignedNodes {
				if u != c.UUID {
					toSelect = append(toSelect, u)
				}
			}
		}

		// select one node without roles to become master
		if len(toSelect) == 1 {
			selected = toSelect[0]
		} else {
			selected = toSelect[rand.Intn(len(toSelect)-1)]
		}

		if err := c.Client.Set("role", selected, masterRole); err != nil {
			return err
		}
		c.Logger.Info("-> Set master to", selected)
		currentRoles[selected] = masterRole
		// Return here, so next time we get called
		// makes sure master is set.
		return nil
	}

	// cycle all empty roles and assign worker roles
	for _, uuid := range unassignedNodes {
		if err := c.Client.Set("role", uuid, workerRole); err != nil {
			c.Logger.Error(err)
			return err
		}
		c.Logger.Infof("-> Set %s to %s", workerRole, uuid)
	}

	c.Logger.Info("Done scheduling")

	return nil
}

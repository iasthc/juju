// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package status

import (
	"encoding/json"
	"fmt"

	"github.com/juju/names/v6"

	"github.com/juju/juju/cmd/juju/storage"
	"github.com/juju/juju/core/instance"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/core/status"
)

type formattedStatus struct {
	Model              modelStatus                        `json:"model"`
	Machines           map[string]machineStatus           `json:"machines"`
	Applications       map[string]applicationStatus       `json:"applications"`
	RemoteApplications map[string]remoteApplicationStatus `json:"application-endpoints,omitempty" yaml:"application-endpoints,omitempty"`
	Offers             map[string]offerStatus             `json:"offers,omitempty" yaml:"offers,omitempty"`
	Relations          []relationStatus                   `json:"-" yaml:"-"`
	Storage            *storage.CombinedStorage           `json:"storage,omitempty" yaml:"storage,omitempty"`
	Controller         *controllerStatus                  `json:"controller,omitempty" yaml:"controller,omitempty"`
}

type formattedMachineStatus struct {
	Model    string                   `json:"model"`
	Machines map[string]machineStatus `json:"machines"`
}

type errorStatus struct {
	StatusError string `json:"status-error" yaml:"status-error"`
}

type modelStatus struct {
	Name             string             `json:"name" yaml:"name"`
	Type             string             `json:"type" yaml:"type"`
	Controller       string             `json:"controller" yaml:"controller"`
	Cloud            string             `json:"cloud" yaml:"cloud"`
	CloudRegion      string             `json:"region,omitempty" yaml:"region,omitempty"`
	Version          string             `json:"version" yaml:"version"`
	AvailableVersion string             `json:"upgrade-available,omitempty" yaml:"upgrade-available,omitempty"`
	Status           statusInfoContents `json:"model-status,omitempty" yaml:"model-status,omitempty"`
}

type controllerStatus struct {
	Timestamp string `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
}

type networkInterface struct {
	IPAddresses    []string `json:"ip-addresses" yaml:"ip-addresses"`
	MACAddress     string   `json:"mac-address" yaml:"mac-address"`
	Gateway        string   `json:"gateway,omitempty" yaml:"gateway,omitempty"`
	DNSNameservers []string `json:"dns-nameservers,omitempty" yaml:"dns-nameservers,omitempty"`
	Space          string   `json:"space,omitempty" yaml:"space,omitempty"`
	IsUp           bool     `json:"is-up" yaml:"is-up"`
}

type machineStatus struct {
	Err                error                         `json:"-" yaml:",omitempty"`
	JujuStatus         statusInfoContents            `json:"juju-status,omitempty" yaml:"juju-status,omitempty"`
	Hostname           string                        `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	DNSName            string                        `json:"dns-name,omitempty" yaml:"dns-name,omitempty"`
	IPAddresses        []string                      `json:"ip-addresses,omitempty" yaml:"ip-addresses,omitempty"`
	InstanceId         instance.Id                   `json:"instance-id,omitempty" yaml:"instance-id,omitempty"`
	DisplayName        string                        `json:"display-name,omitempty" yaml:"display-name,omitempty"`
	MachineStatus      statusInfoContents            `json:"machine-status,omitempty" yaml:"machine-status,omitempty"`
	ModificationStatus statusInfoContents            `json:"modification-status,omitempty" yaml:"modification-status,omitempty"`
	Base               *formattedBase                `json:"base,omitempty" yaml:"base,omitempty"`
	Id                 string                        `json:"-" yaml:"-"`
	NetworkInterfaces  map[string]networkInterface   `json:"network-interfaces,omitempty" yaml:"network-interfaces,omitempty"`
	Containers         map[string]machineStatus      `json:"containers,omitempty" yaml:"containers,omitempty"`
	Constraints        string                        `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Hardware           string                        `json:"hardware,omitempty" yaml:"hardware,omitempty"`
	HAStatus           string                        `json:"controller-member-status,omitempty" yaml:"controller-member-status,omitempty"`
	HAPrimary          bool                          `json:"ha-primary,omitempty" yaml:"ha-primary,omitempty"`
	LXDProfiles        map[string]lxdProfileContents `json:"lxd-profiles,omitempty" yaml:"lxd-profiles,omitempty"`
}

// A goyaml bug means we can't declare these types
// locally to the GetYAML methods.
type machineStatusNoMarshal machineStatus

func (s machineStatus) MarshalJSON() ([]byte, error) {
	if s.Err != nil {
		return json.Marshal(errorStatus{s.Err.Error()})
	}
	return json.Marshal(machineStatusNoMarshal(s))
}

func (s machineStatus) MarshalYAML() (interface{}, error) {
	if s.Err != nil {
		return errorStatus{s.Err.Error()}, nil
	}
	return machineStatusNoMarshal(s), nil
}

// machineName returns the InstanceId, unless DisplayName is set.
func (s machineStatus) machineName() string {
	if s.DisplayName == "" {
		return string(s.InstanceId)
	}
	return s.DisplayName
}

// LXDProfile holds status info about a LXDProfile
type lxdProfileContents struct {
	Config      map[string]string            `json:"config" yaml:"config"`
	Description string                       `json:"description" yaml:"description"`
	Devices     map[string]map[string]string `json:"devices" yaml:"devices"`
}

type applicationStatus struct {
	Err              error                                  `json:"-" yaml:",omitempty"`
	Charm            string                                 `json:"charm" yaml:"charm"`
	Base             *formattedBase                         `json:"base,omitempty" yaml:"base,omitempty"`
	CharmOrigin      string                                 `json:"charm-origin" yaml:"charm-origin"`
	CharmName        string                                 `json:"charm-name" yaml:"charm-name"`
	CharmRev         int                                    `json:"charm-rev" yaml:"charm-rev"`
	CharmChannel     string                                 `json:"charm-channel,omitempty" yaml:"charm-channel,omitempty"`
	CharmVersion     string                                 `json:"charm-version,omitempty" yaml:"charm-version,omitempty"`
	CharmProfile     string                                 `json:"charm-profile,omitempty" yaml:"charm-profile,omitempty"`
	CanUpgradeTo     string                                 `json:"can-upgrade-to,omitempty" yaml:"can-upgrade-to,omitempty"`
	Scale            int                                    `json:"scale,omitempty" yaml:"scale,omitempty"`
	ProviderId       string                                 `json:"provider-id,omitempty" yaml:"provider-id,omitempty"`
	Address          string                                 `json:"address,omitempty" yaml:"address,omitempty"`
	Exposed          bool                                   `json:"exposed" yaml:"exposed"`
	Life             string                                 `json:"life,omitempty" yaml:"life,omitempty"`
	StatusInfo       statusInfoContents                     `json:"application-status,omitempty" yaml:"application-status"`
	Relations        map[string][]applicationStatusRelation `json:"relations,omitempty" yaml:"relations,omitempty"`
	SubordinateTo    []string                               `json:"subordinate-to,omitempty" yaml:"subordinate-to,omitempty"`
	Units            map[string]unitStatus                  `json:"units,omitempty" yaml:"units,omitempty"`
	Version          string                                 `json:"version,omitempty" yaml:"version,omitempty"`
	EndpointBindings map[string]string                      `json:"endpoint-bindings,omitempty" yaml:"endpoint-bindings,omitempty"`
}

type applicationStatusRelation struct {
	RelatedApplicationName string `json:"related-application,omitempty" yaml:"related-application,omitempty"`
	Interface              string `json:"interface,omitempty" yaml:"interface,omitempty"`
	Scope                  string `json:"scope,omitempty" yaml:"scope,omitempty"`
}

type formattedBase struct {
	Name    string `json:"name" yaml:"name"`
	Channel string `json:"channel" yaml:"channel"`
}

type applicationStatusNoMarshal applicationStatus

func (s applicationStatus) MarshalJSON() ([]byte, error) {
	if s.Err != nil {
		return json.Marshal(errorStatus{s.Err.Error()})
	}
	return json.Marshal(applicationStatusNoMarshal(s))
}

func (s applicationStatus) MarshalYAML() (interface{}, error) {
	if s.Err != nil {
		return errorStatus{s.Err.Error()}, nil
	}
	return applicationStatusNoMarshal(s), nil
}

type remoteEndpoint struct {
	Name      string `json:"-" yaml:"-"`
	Interface string `json:"interface" yaml:"interface"`
	Role      string `json:"role" yaml:"role"`
}

type remoteApplicationStatus struct {
	Err        error                     `json:"-" yaml:",omitempty"`
	OfferURL   string                    `json:"url" yaml:"url"`
	Endpoints  map[string]remoteEndpoint `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Life       string                    `json:"life,omitempty" yaml:"life,omitempty"`
	StatusInfo statusInfoContents        `json:"application-status,omitempty" yaml:"application-status"`
	Relations  map[string][]string       `json:"relations,omitempty" yaml:"relations,omitempty"`
}

type remoteApplicationStatusNoMarshal remoteApplicationStatus

func (s remoteApplicationStatus) MarshalJSON() ([]byte, error) {
	if s.Err != nil {
		return json.Marshal(errorStatus{s.Err.Error()})
	}
	return json.Marshal(remoteApplicationStatusNoMarshal(s))
}

func (s remoteApplicationStatus) MarshalYAML() (interface{}, error) {
	if s.Err != nil {
		return errorStatus{s.Err.Error()}, nil
	}
	return remoteApplicationStatusNoMarshal(s), nil
}

type offerStatusNoMarshal offerStatus

type offerStatus struct {
	Err                  error                     `json:"-" yaml:",omitempty"`
	OfferName            string                    `json:"-" yaml:",omitempty"`
	ApplicationName      string                    `json:"application" yaml:"application"`
	CharmURL             string                    `json:"charm,omitempty" yaml:"charm,omitempty"`
	TotalConnectedCount  int                       `json:"total-connected-count,omitempty" yaml:"total-connected-count,omitempty"`
	ActiveConnectedCount int                       `json:"active-connected-count,omitempty" yaml:"active-connected-count,omitempty"`
	Endpoints            map[string]remoteEndpoint `json:"endpoints" yaml:"endpoints"`
}

func (s offerStatus) MarshalJSON() ([]byte, error) {
	if s.Err != nil {
		return json.Marshal(errorStatus{s.Err.Error()})
	}
	return json.Marshal(offerStatusNoMarshal(s))
}

func (s offerStatus) MarshalYAML() (interface{}, error) {
	if s.Err != nil {
		return errorStatus{s.Err.Error()}, nil
	}
	return offerStatusNoMarshal(s), nil
}

type unitStatus struct {
	// New Juju Health Status fields.
	WorkloadStatusInfo statusInfoContents `json:"workload-status,omitempty" yaml:"workload-status,omitempty"`
	JujuStatusInfo     statusInfoContents `json:"juju-status,omitempty" yaml:"juju-status,omitempty"`

	Leader        bool                  `json:"leader,omitempty" yaml:"leader,omitempty"`
	Charm         string                `json:"upgrading-from,omitempty" yaml:"upgrading-from,omitempty"`
	Machine       string                `json:"machine,omitempty" yaml:"machine,omitempty"`
	OpenedPorts   []string              `json:"open-ports,omitempty" yaml:"open-ports,omitempty"`
	PublicAddress string                `json:"public-address,omitempty" yaml:"public-address,omitempty"`
	Address       string                `json:"address,omitempty" yaml:"address,omitempty"`
	ProviderId    string                `json:"provider-id,omitempty" yaml:"provider-id,omitempty"`
	Subordinates  map[string]unitStatus `json:"subordinates,omitempty" yaml:"subordinates,omitempty"`
}

func (s *formattedStatus) applicationScale(name string) (string, bool) {
	// The current unit count are units that are either in Idle or Executing status.
	// In other words, units that are active and available.
	currentUnitCount := 0
	desiredUnitCount := 0

	app := s.Applications[name]
	match := func(u unitStatus) {
		desiredUnitCount++
		if u.WorkloadStatusInfo.Current == status.Terminated {
			return
		}
		switch u.JujuStatusInfo.Current {
		case status.Executing, status.Idle, status.Running:
			currentUnitCount++
		}
	}
	// If the app is subordinate to other units, then this is a subordinate charm.
	if len(app.SubordinateTo) > 0 {
		for _, a := range s.Applications {
			for _, u := range a.Units {
				for sub, subStatus := range u.Subordinates {
					if subAppName, _ := names.UnitApplication(sub); subAppName == name {
						match(subStatus)
					}
				}
			}
		}
	} else {
		for _, u := range app.Units {
			match(u)
		}
	}
	if s.Model.Type == string(coremodel.CAAS) {
		desiredUnitCount = app.Scale
	}
	if currentUnitCount == desiredUnitCount {
		return fmt.Sprint(currentUnitCount), false
	}
	return fmt.Sprintf("%d/%d", currentUnitCount, desiredUnitCount), true
}

type statusInfoContents struct {
	Err     error         `json:"-" yaml:",omitempty"`
	Current status.Status `json:"current,omitempty" yaml:"current,omitempty"`
	Message string        `json:"message,omitempty" yaml:"message,omitempty"`
	Reason  string        `json:"reason,omitempty" yaml:"reason,omitempty"`
	Since   string        `json:"since,omitempty" yaml:"since,omitempty"`
	Version string        `json:"version,omitempty" yaml:"version,omitempty"`
	Life    string        `json:"life,omitempty" yaml:"life,omitempty"`
}

type statusInfoContentsNoMarshal statusInfoContents

func (s statusInfoContents) MarshalJSON() ([]byte, error) {
	if s.Err != nil {
		return json.Marshal(errorStatus{s.Err.Error()})
	}
	return json.Marshal(statusInfoContentsNoMarshal(s))
}

func (s statusInfoContents) MarshalYAML() (interface{}, error) {
	if s.Err != nil {
		return errorStatus{s.Err.Error()}, nil
	}
	return statusInfoContentsNoMarshal(s), nil
}

type unitStatusNoMarshal unitStatus

func (s unitStatus) MarshalJSON() ([]byte, error) {
	if s.WorkloadStatusInfo.Err != nil {
		return json.Marshal(errorStatus{s.WorkloadStatusInfo.Err.Error()})
	}
	return json.Marshal(unitStatusNoMarshal(s))
}

func (s unitStatus) MarshalYAML() (interface{}, error) {
	if s.WorkloadStatusInfo.Err != nil {
		return errorStatus{s.WorkloadStatusInfo.Err.Error()}, nil
	}
	return unitStatusNoMarshal(s), nil
}

type relationStatus struct {
	Provider  string
	Requirer  string
	Interface string
	Type      string
	Status    string
	Message   string
}

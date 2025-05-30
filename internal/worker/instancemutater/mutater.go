// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instancemutater

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/juju/clock"
	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/names/v6"
	"github.com/juju/worker/v4"

	"github.com/juju/juju/api/agent/instancemutater"
	"github.com/juju/juju/core/instance"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/logger"
	"github.com/juju/juju/core/lxdprofile"
	"github.com/juju/juju/core/status"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/internal/wrench"
	"github.com/juju/juju/rpc/params"
)

//go:generate go run go.uber.org/mock/mockgen -typed -package mocks -destination mocks/mutatercontext_mock.go github.com/juju/juju/internal/worker/instancemutater MutaterContext

// lifetimeContext was extracted to allow the various Context clients to get
// the benefits of the catacomb encapsulating everything that should happen
// here. A clean implementation would almost certainly not need this.
type lifetimeContext interface {
	KillWithError(error)

	add(worker.Worker) error
	dying() <-chan struct{}
	errDying() error
}

type MachineContext interface {
	lifetimeContext
	getBroker() environs.LXDProfiler
	getRequiredLXDProfiles(string) []string
}

type MutaterMachine struct {
	context    MachineContext
	logger     logger.Logger
	machineApi instancemutater.MutaterMachine
	id         string
}

type MutaterContext interface {
	MachineContext
	newMachineContext() MachineContext
	getMachine(ctx context.Context, tag names.MachineTag) (instancemutater.MutaterMachine, error)
}

type mutater struct {
	context     MutaterContext
	logger      logger.Logger
	wg          *sync.WaitGroup
	machines    map[names.MachineTag]chan struct{}
	machineDead chan instancemutater.MutaterMachine
}

func (m *mutater) startMachines(ctx context.Context, tags []names.MachineTag) error {
	for _, tag := range tags {
		select {
		case <-m.context.dying():
			return m.context.errDying()
		default:
		}
		m.logger.Tracef(ctx, "received tag %q", tag.String())
		if ch := m.machines[tag]; ch == nil {
			// First time we receive the tag, setup watchers.
			api, err := m.context.getMachine(ctx, tag)
			if err != nil {
				return errors.Trace(err)
			}
			id := api.Tag().Id()

			// Ensure we do not watch any containers that aren't LXD.
			containerType, err := api.ContainerType(ctx)
			if err != nil {
				return errors.Trace(err)
			}
			if containerType != instance.LXD {
				m.logger.Tracef(ctx, "ignoring %q container machine-%s", containerType, id)
				continue
			}

			profileChangeWatcher, err := api.WatchLXDProfileVerificationNeeded(ctx)
			if err != nil {
				if errors.Is(err, errors.NotSupported) {
					m.logger.Tracef(ctx, "ignoring manual machine-%s", id)
					continue
				}
				return errors.Annotatef(err, "failed to start watching application lxd profiles for machine-%s", id)
			}

			ch = make(chan struct{})
			m.machines[tag] = ch

			machine := MutaterMachine{
				context:    m.context.newMachineContext(),
				logger:     m.logger,
				machineApi: api,
				id:         id,
			}

			m.wg.Add(1)
			go runMachine(machine, profileChangeWatcher, ch, m.machineDead, func() { m.wg.Done() })
		} else {
			// We've received this tag before, therefore
			// the machine has been removed from the model
			// cache and no longer needed
			ch <- struct{}{}
		}
	}
	return nil
}

func runMachine(
	machine MutaterMachine,
	profileChangeWatcher watcher.NotifyWatcher,
	removed <-chan struct{}, died chan<- instancemutater.MutaterMachine, cleanup func(),
) {
	defer cleanup()
	defer func() {
		// We can't just send on the dead channel because the
		// central loop might be trying to write to us on the
		// removed channel.
		for {
			select {
			case <-machine.context.dying():
				return
			case died <- machine.machineApi:
				return
			case <-removed:
			}
		}
	}()

	if err := machine.context.add(profileChangeWatcher); err != nil {
		machine.context.KillWithError(err)
		return
	}
	if err := machine.watchProfileChangesLoop(removed, profileChangeWatcher); err != nil {
		machine.context.KillWithError(err)
	}
}

// watchProfileChanges, any error returned will cause the worker to restart.
func (m MutaterMachine) watchProfileChangesLoop(removed <-chan struct{}, profileChangeWatcher watcher.NotifyWatcher) error {
	m.logger.Tracef(context.TODO(), "watching change on MutaterMachine %s", m.id)
	for {
		select {
		case <-m.context.dying():
			return m.context.errDying()
		case <-profileChangeWatcher.Changes():
			info, err := m.machineApi.CharmProfilingInfo(context.TODO())
			if err != nil {
				// If the machine is not provisioned then we need to wait for
				// new changes from the watcher.
				if params.IsCodeNotProvisioned(errors.Cause(err)) {
					m.logger.Tracef(context.TODO(), "got not provisioned machine-%s on charm profiling info, wait for another change", m.id)
					continue
				}
				return errors.Trace(err)
			}
			if err = m.processMachineProfileChanges(context.TODO(), info); err != nil && errors.Is(err, errors.NotValid) {
				// Return to stop mutating the machine, but no need to restart
				// the worker.
				return nil
			} else if err != nil {
				return errors.Trace(err)
			}
		case <-removed:
			if err := m.machineApi.Refresh(context.TODO()); err != nil {
				return errors.Trace(err)
			}
			if m.machineApi.Life() == life.Dead {
				return nil
			}
		}
	}
}

func (m MutaterMachine) processMachineProfileChanges(ctx context.Context, info *instancemutater.UnitProfileInfo) error {
	if info == nil || (len(info.CurrentProfiles) == 0 && len(info.ProfileChanges) == 0) {
		// no changes to be made, return now.
		return nil
	}

	if err := m.machineApi.Refresh(ctx); err != nil {
		return err
	}
	if m.machineApi.Life() == life.Dead {
		return errors.NotValidf("machine %q", m.id)
	}

	// Set the modification status to idle, that way we have a baseline for
	// future changes.
	if err := m.machineApi.SetModificationStatus(ctx, status.Idle, "", nil); err != nil {
		return errors.Annotatef(err, "cannot set status for machine %q modification status", m.id)
	}

	report := func(retErr error) error {
		if retErr != nil {
			m.logger.Errorf(ctx, "cannot upgrade machine-%s lxd profiles: %s", m.id, retErr.Error())
			if err := m.machineApi.SetModificationStatus(ctx, status.Error, fmt.Sprintf("cannot upgrade machine's lxd profile: %s", retErr.Error()), nil); err != nil {
				m.logger.Errorf(ctx, "cannot set modification status of machine %q error: %v", m.id, err)
			}
		} else {
			if err := m.machineApi.SetModificationStatus(ctx, status.Applied, "", nil); err != nil {
				m.logger.Errorf(ctx, "cannot reset modification status of machine %q applied: %v", m.id, err)
			}
		}
		return retErr
	}

	// Convert info.ProfileChanges into a struct which can be used to
	// add or remove profiles from a machine.  Use it to create a list
	// of expected profiles.
	post, err := m.gatherProfileData(info)
	if err != nil {
		return report(errors.Annotatef(err, "%s", m.id))
	}

	expectedProfiles := m.context.getRequiredLXDProfiles(info.ModelName)
	for _, p := range post {
		if p.Profile != nil {
			expectedProfiles = append(expectedProfiles, p.Name)
		}
	}

	verified, currentProfiles, err := m.verifyCurrentProfiles(string(info.InstanceId), expectedProfiles)
	if err != nil {
		return report(errors.Annotatef(err, "%s", m.id))
	}
	if verified {
		m.logger.Infof(ctx, "no changes necessary to machine-%s lxd profiles (%v)", m.id, expectedProfiles)
		return report(m.machineApi.SetCharmProfiles(ctx, lxdprofile.FilterLXDProfileNames(currentProfiles)))
	}

	// Adding a wrench to test charm not running hooks before profile can be applied.
	// Do not bother for the default or model profile.  We're not interested in non
	// charm profiles.
	if wrench.IsActive("instance-mutater", "disable-apply-lxdprofile") && len(expectedProfiles) > 1 {
		m.logger.Warningf(ctx, "waiting 3 minutes to apply lxd profiles %q due to wrench in the works", strings.Join(expectedProfiles, ", "))
		select {
		case <-clock.WallClock.After(3 * time.Minute):
			m.logger.Warningf(ctx, "continue with apply lxd profiles")
		}
	}

	m.logger.Infof(ctx, "machine-%s (%s) assign lxd profiles %q, %#v", m.id, string(info.InstanceId), expectedProfiles, post)
	broker := m.context.getBroker()
	currentProfiles, err = broker.AssignLXDProfiles(string(info.InstanceId), expectedProfiles, post)
	if err != nil {
		m.logger.Errorf(ctx, "failure to assign lxd profiles %s to machine-%s: %s", expectedProfiles, m.id, err)
		return report(err)
	}

	return report(m.machineApi.SetCharmProfiles(ctx, lxdprofile.FilterLXDProfileNames(currentProfiles)))
}

func (m MutaterMachine) gatherProfileData(info *instancemutater.UnitProfileInfo) ([]lxdprofile.ProfilePost, error) {
	var result []lxdprofile.ProfilePost
	for _, pu := range info.ProfileChanges {
		oldName, err := lxdprofile.MatchProfileNameByAppName(info.CurrentProfiles, pu.ApplicationName)
		if err != nil {
			return nil, err
		}
		if pu.Profile.Empty() && oldName == "" {
			// There is no new Profile and no Profile for this application applied
			// already, move on.  A charm without an lxd profile.
			continue
		}
		name := lxdprofile.Name(info.ModelName, pu.ApplicationName, pu.Revision)
		if oldName != "" && name != oldName {
			// add the old profile name to the result, so the profile can
			// be deleted from the lxd server.
			result = append(result, lxdprofile.ProfilePost{Name: oldName})
		}
		add := lxdprofile.ProfilePost{Name: name}
		// should not happen, but you never know.
		if !pu.Profile.Empty() {
			// We make a copy since the loop var keeps the same pointer.
			p := pu.Profile
			add.Profile = &p
		}
		result = append(result, add)
	}
	return result, nil
}

func (m MutaterMachine) verifyCurrentProfiles(instID string, expectedProfiles []string) (bool, []string, error) {
	broker := m.context.getBroker()
	obtainedProfiles, err := broker.LXDProfileNames(instID)
	if err != nil {
		return false, nil, err
	}

	if len(obtainedProfiles) == 0 && len(expectedProfiles) == 0 {
		return true, obtainedProfiles, nil
	} else if len(obtainedProfiles) != len(expectedProfiles) {
		return false, obtainedProfiles, nil
	}

	obtainedSet := set.NewStrings(obtainedProfiles...)
	expectedSet := set.NewStrings(expectedProfiles...)

	return obtainedSet.Difference(expectedSet).Size() == 0, obtainedProfiles, nil
}

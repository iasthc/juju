// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpserver_test

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/juju/clock/testclock"
	"github.com/juju/errors"
	"github.com/juju/pubsub/v2"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/worker/v4"
	"github.com/juju/worker/v4/dependency"
	dt "github.com/juju/worker/v4/dependency/testing"
	"github.com/juju/worker/v4/workertest"
	"golang.org/x/crypto/acme/autocert"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/apiserverhttp"
	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/logger"
	autocertcacheservice "github.com/juju/juju/domain/autocert/service"
	controllerconfigservice "github.com/juju/juju/domain/controllerconfig/service"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	"github.com/juju/juju/internal/pki"
	pkitest "github.com/juju/juju/internal/pki/test"
	"github.com/juju/juju/internal/services"
	"github.com/juju/juju/internal/worker/httpserver"
	"github.com/juju/juju/state"
)

type ManifoldSuite struct {
	testing.IsolationSuite

	authority              pki.Authority
	config                 httpserver.ManifoldConfig
	manifold               dependency.Manifold
	getter                 dependency.Getter
	state                  stubStateTracker
	hub                    *pubsub.StructuredHub
	mux                    *apiserverhttp.Mux
	clock                  *testclock.Clock
	prometheusRegisterer   stubPrometheusRegisterer
	tlsConfig              *tls.Config
	controllerConfig       controller.Config
	domainServices         services.DomainServices
	autocertCacheGetter    *autocertcacheservice.Service
	controllerConfigGetter *controllerconfigservice.WatchableService

	stub testing.Stub
}

var _ = gc.Suite(&ManifoldSuite{})

func (s *ManifoldSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	authority, err := pkitest.NewTestAuthority()
	c.Assert(err, jc.ErrorIsNil)
	s.authority = authority

	s.mux = &apiserverhttp.Mux{}
	s.hub = pubsub.NewStructuredHub(nil)
	s.clock = testclock.NewClock(time.Now())
	s.prometheusRegisterer = stubPrometheusRegisterer{}
	s.tlsConfig = &tls.Config{}
	s.controllerConfig = map[string]interface{}{
		"api-port":            1024,
		"controller-api-port": 2048,
		"api-port-open-delay": "5s",
	}

	s.autocertCacheGetter = &autocertcacheservice.Service{}
	s.controllerConfigGetter = &controllerconfigservice.WatchableService{}
	s.domainServices = stubDomainServices{
		controllerConfigGetter: s.controllerConfigGetter,
		autocertCacheGetter:    s.autocertCacheGetter,
	}
	s.stub.ResetCalls()

	s.getter = s.newGetter(nil)
	s.config = httpserver.ManifoldConfig{
		AgentName:            "machine-42",
		AuthorityName:        "authority",
		HubName:              "hub",
		StateName:            "state",
		DomainServicesName:   "domain-services",
		MuxName:              "mux",
		APIServerName:        "api-server",
		Clock:                s.clock,
		PrometheusRegisterer: &s.prometheusRegisterer,
		MuxShutdownWait:      1 * time.Minute,
		LogDir:               "log-dir",
		GetControllerConfig:  s.getControllerConfig,
		NewTLSConfig:         s.newTLSConfig,
		NewWorker:            s.newWorker,
		Logger:               loggertesting.WrapCheckLog(c),
	}
	s.manifold = httpserver.Manifold(s.config)
	s.state = stubStateTracker{
		pool:   &state.StatePool{},
		system: &state.State{},
	}
}

func (s *ManifoldSuite) newGetter(overlay map[string]interface{}) dependency.Getter {
	resources := map[string]interface{}{
		"authority":       s.authority,
		"state":           &s.state,
		"hub":             s.hub,
		"mux":             s.mux,
		"api-server":      nil,
		"domain-services": s.domainServices,
	}
	for k, v := range overlay {
		resources[k] = v
	}
	return dt.StubGetter(resources)
}

func (s *ManifoldSuite) getControllerConfig(_ context.Context, getter httpserver.ControllerConfigGetter) (controller.Config, error) {
	s.stub.MethodCall(s, "GetControllerConfig", getter)
	if err := s.stub.NextErr(); err != nil {
		return nil, err
	}
	return s.controllerConfig, nil
}

func (s *ManifoldSuite) newTLSConfig(
	dnsName string,
	serverURL string,
	cache autocert.Cache,
	_ httpserver.SNIGetterFunc,
	_ logger.Logger,
) *tls.Config {
	s.stub.MethodCall(s, "NewTLSConfig", dnsName)
	return s.tlsConfig
}

func (s *ManifoldSuite) newWorker(config httpserver.Config) (worker.Worker, error) {
	s.stub.MethodCall(s, "NewWorker", config)
	if err := s.stub.NextErr(); err != nil {
		return nil, err
	}
	return worker.NewRunner(worker.RunnerParams{
		Name: "httpserver",
	})
}

var expectedInputs = []string{
	"authority",
	"state",
	"mux",
	"hub",
	"api-server",
	"domain-services",
}

func (s *ManifoldSuite) TestInputs(c *gc.C) {
	c.Assert(s.manifold.Inputs, jc.SameContents, expectedInputs)
}

func (s *ManifoldSuite) TestMissingInputs(c *gc.C) {
	for _, input := range expectedInputs {
		getter := s.newGetter(map[string]interface{}{
			input: dependency.ErrMissing,
		})
		_, err := s.manifold.Start(context.Background(), getter)
		c.Assert(errors.Cause(err), gc.Equals, dependency.ErrMissing)
	}
}

func (s *ManifoldSuite) TestStart(c *gc.C) {
	w := s.startWorkerClean(c)
	workertest.CleanKill(c, w)

	s.stub.CheckCallNames(c, "GetControllerConfig", "NewTLSConfig", "NewWorker")
	newWorkerArgs := s.stub.Calls()[2].Args
	c.Assert(newWorkerArgs, gc.HasLen, 1)
	c.Assert(newWorkerArgs[0], gc.FitsTypeOf, httpserver.Config{})
	config := newWorkerArgs[0].(httpserver.Config)

	c.Assert(config, jc.DeepEquals, httpserver.Config{
		AgentName:            "machine-42",
		Clock:                s.clock,
		PrometheusRegisterer: &s.prometheusRegisterer,
		Hub:                  s.hub,
		TLSConfig:            s.tlsConfig,
		Mux:                  s.mux,
		APIPort:              1024,
		APIPortOpenDelay:     5 * time.Second,
		ControllerAPIPort:    2048,
		MuxShutdownWait:      1 * time.Minute,
		LogDir:               "log-dir",
		Logger:               s.config.Logger,
	})
}

func (s *ManifoldSuite) TestValidate(c *gc.C) {
	type test struct {
		f      func(*httpserver.ManifoldConfig)
		expect string
	}
	tests := []test{{
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.AgentName = "" },
		expect: "empty AgentName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.AuthorityName = "" },
		expect: "empty AuthorityName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.StateName = "" },
		expect: "empty StateName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.DomainServicesName = "" },
		expect: "empty DomainServicesName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.MuxName = "" },
		expect: "empty MuxName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.MuxShutdownWait = 0 },
		expect: "MuxShutdownWait 0s not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.LogDir = "" },
		expect: "empty LogDir not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.APIServerName = "" },
		expect: "empty APIServerName not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.PrometheusRegisterer = nil },
		expect: "nil PrometheusRegisterer not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.GetControllerConfig = nil },
		expect: "nil GetControllerConfig not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.NewTLSConfig = nil },
		expect: "nil NewTLSConfig not valid",
	}, {
		f:      func(cfg *httpserver.ManifoldConfig) { cfg.NewWorker = nil },
		expect: "nil NewWorker not valid",
	}}
	for i, test := range tests {
		c.Logf("test #%d (%s)", i, test.expect)
		config := s.config
		test.f(&config)
		manifold := httpserver.Manifold(config)
		w, err := manifold.Start(context.Background(), s.getter)
		workertest.CheckNilOrKill(c, w)
		c.Check(err, gc.ErrorMatches, test.expect)
	}
}

func (s *ManifoldSuite) startWorkerClean(c *gc.C) worker.Worker {
	w, err := s.manifold.Start(context.Background(), s.getter)
	c.Assert(err, jc.ErrorIsNil)
	workertest.CheckAlive(c, w)
	return w
}

type stubDomainServices struct {
	services.DomainServices
	controllerConfigGetter *controllerconfigservice.WatchableService
	autocertCacheGetter    *autocertcacheservice.Service
}

func (s stubDomainServices) AutocertCache() *autocertcacheservice.Service {
	return s.autocertCacheGetter
}

func (s stubDomainServices) ControllerConfig() *controllerconfigservice.WatchableService {
	return s.controllerConfigGetter
}

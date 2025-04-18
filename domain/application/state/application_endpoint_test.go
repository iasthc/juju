// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"context"
	"database/sql"

	"github.com/canonical/sqlair"
	"github.com/juju/clock"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	coreapplication "github.com/juju/juju/core/application"
	applicationtesting "github.com/juju/juju/core/application/testing"
	corecharm "github.com/juju/juju/core/charm"
	charmtesting "github.com/juju/juju/core/charm/testing"
	"github.com/juju/juju/core/network"
	applicationerrors "github.com/juju/juju/domain/application/errors"
	"github.com/juju/juju/internal/errors"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	"github.com/juju/juju/internal/uuid"
)

// applicationEndpointStateSuite defines the testing suite for managing
// application endpoint state operations.
//
// It embeds baseSuite and provides constants and state for test scenarios.
type applicationEndpointStateSuite struct {
	baseSuite

	appID     coreapplication.ID
	charmUUID corecharm.ID

	state *State
}

var _ = gc.Suite(&applicationEndpointStateSuite{})

// SetUpTest sets up the testing environment by initializing the suite's state
// and arranging the required database context:
//   - One charm
//   - One application
func (s *applicationEndpointStateSuite) SetUpTest(c *gc.C) {
	s.baseSuite.SetUpTest(c)

	s.state = NewState(s.TxnRunnerFactory(), clock.WallClock, loggertesting.WrapCheckLog(c))

	// Arrange suite context, same for all tests:
	s.appID = applicationtesting.GenApplicationUUID(c)
	s.charmUUID = charmtesting.GenCharmID(c)

	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		var err error
		_, err = tx.ExecContext(ctx, `
INSERT INTO charm (uuid, reference_name, source_id) 
VALUES (?, 'foo', 0)`, s.charmUUID)
		if err != nil {
			return errors.Capture(err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO application (uuid, charm_uuid, name, life_id, space_uuid)
VALUES (?,?,?,0,?)`, s.appID, s.charmUUID, "foo", network.AlphaSpaceId)
		if err != nil {
			return errors.Capture(err)
		}

		return nil
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Arrange suite) Failed to setup test suite: %v", err))
}

// TestInsertApplicationNoCharmRelation validates behavior when inserting
// application endpoints without a charm relation.
//
// Ensures no relation endpoints are created and no errors occur during the operation.
func (s *applicationEndpointStateSuite) TestInsertApplicationNoCharmRelation(c *gc.C) {
	// Arrange: No relation
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)

	// Act: noop, no error
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
		})
	})

	// Assert: Shouldn't have any relation endpoint, default space not updated
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, network.AlphaSpaceName)
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{})
}

// TestInsertApplicationNoCharmRelation validates behavior when inserting
// application endpoints without a charm relation.
//
// Ensures no relation endpoints are created and no errors occur during the operation,
// while the default enpoint is correctly set
func (s *applicationEndpointStateSuite) TestInsertApplicationNoCharmRelationWithDefaultEndpoint(c *gc.C) {
	// Arrange: No relation
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	bindings := map[string]network.SpaceName{
		"": s.addSpace(c, "beta"),
	}

	// Act: noop, no error
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: Shouldn't have any relation endpoint, but default should be updated
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, "beta")
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{})
}

// TestInsertApplicationNoBindings tests the insertion of application
// endpoints with no bindings
func (s *applicationEndpointStateSuite) TestInsertApplicationNoBindings(c *gc.C) {
	// Arrange: One expected relation, one extra endpoint, no binding
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	relUUID := s.addRelation(c, "default")
	extraUUID := s.addExtraBinding(c, "extra")

	// Act: Charm relation will create application endpoint bounded to the default space (alpha)
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
		})
	})

	// Assert: Should have
	//  - default space not updated.
	//  - an application endpoint without spacename,
	//  - an application extra endpoint without spacename,
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, network.AlphaSpaceName)
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: relUUID,
		},
	})
	c.Check(s.fetchApplicationExtraEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: extraUUID,
		},
	})
}

// TestInsertApplicationEndpointDefaultedSpace verifies the insertion of
// application endpoints while setting the default space
func (s *applicationEndpointStateSuite) TestInsertApplicationEndpointDefaultedSpace(c *gc.C) {
	// Arrange:
	// - One expected relation, one expected endpoint
	// - override default space to beta
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	relUUID := s.addRelation(c, "default")
	extraUUID := s.addExtraBinding(c, "extra")
	bindings := map[string]network.SpaceName{
		"": s.addSpace(c, "beta"),
	}

	// Act: Charm relation will create application endpoint bounded to the default space (beta)
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: Should have
	//  - default space updated to beta.
	//  - an application endpoint without spacename,
	//  - an application extra endpoint without spacename,
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, "beta")
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: relUUID,
		},
	})
	c.Check(s.fetchApplicationExtraEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: extraUUID,
		},
	})
}

// TestInsertApplicationEndpointBindOneToBeta verifies that an application
// endpoint can be correctly bound to a specific space.
func (s *applicationEndpointStateSuite) TestInsertApplicationEndpointBindOneToBeta(c *gc.C) {
	// Arrange:
	// - two expected relation
	// - two expected extra endpoint
	// - one of both are bound with a specific space (beta)
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	relUUID := s.addRelation(c, "default")
	boundUUID := s.addRelation(c, "bound")
	extraUUID := s.addExtraBinding(c, "extra")
	boundExtraUUID := s.addExtraBinding(c, "bound-extra")
	bindings := map[string]network.SpaceName{
		"bound":       s.addSpace(c, "beta"),
		"bound-extra": s.addSpace(c, "beta-extra"),
	}

	// Act: Charm relation will create application endpoint bounded to the specified space (beta)
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: Should have
	//  - default space not updated.
	//  - two application endpoint one without spacename, one bound to beta
	//  - two application extra endpoint one without spacename, one bound to beta-extra
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, network.AlphaSpaceName)
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: relUUID,
		},
		{
			charmRelationUUID: boundUUID,
			spaceName:         "beta",
		},
	})
	c.Check(s.fetchApplicationExtraEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: extraUUID,
		},
		{
			charmRelationUUID: boundExtraUUID,
			spaceName:         "beta-extra",
		},
	})
}

// TestInsertApplicationEndpointBindOneToBetaDefaultedGamma tests the insertion
// of application endpoints with space bindings.
func (s *applicationEndpointStateSuite) TestInsertApplicationEndpointBindOneToBetaDefaultedGamma(c *gc.C) {
	// Arrange:
	// - two expected relation and extra endpoint
	// - override default space
	// - bind one relation to a specific space
	// - bind one extra relation to a specific space
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	relUUID := s.addRelation(c, "default")
	boundUUID := s.addRelation(c, "bound")
	extraUUID := s.addExtraBinding(c, "extra")
	boundExtraUUID := s.addExtraBinding(c, "bound-extra")
	beta := s.addSpace(c, "beta")
	bindings := map[string]network.SpaceName{
		"":            s.addSpace(c, "gamma"),
		"bound":       beta,
		"bound-extra": beta,
	}

	// Act: Charm relation will create application endpoint bounded to either the defaulted space
	// or the specified one
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: Should have
	//  - default space updated to gamma
	//  - two application endpoint one without spacename, one bound to beta
	//  - two application extra endpoint one without spacename, one bound to beta
	c.Assert(err, jc.ErrorIsNil)
	c.Check(s.getApplicationDefaultSpace(c), gc.Equals, "gamma")
	c.Check(s.fetchApplicationEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: relUUID,
		},
		{
			charmRelationUUID: boundUUID,
			spaceName:         "beta",
		},
	})

	c.Check(s.fetchApplicationExtraEndpoints(c), jc.DeepEquals, []applicationEndpoint{
		{
			charmRelationUUID: extraUUID,
		},
		{
			charmRelationUUID: boundExtraUUID,
			spaceName:         "beta",
		},
	})
}

// TestInsertApplicationEndpointUnknownSpace verifies the behavior of inserting
// application endpoints with an unknown space.
//
// Ensures that an error is returned
func (s *applicationEndpointStateSuite) TestInsertApplicationEndpointUnknownSpace(c *gc.C) {
	// Arrange:
	// - One expected relation
	// - bind with an unknown space
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	s.addRelation(c, "default")
	bindings := map[string]network.SpaceName{
		"": "unknown",
	}

	// Act: Charm relation will create application endpoint bounded to the default space (alpha)
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: should fail because unknown is not a valid space
	c.Assert(err, jc.ErrorIs, applicationerrors.SpaceNotFound)
}

// TestInsertApplicationEndpointUnknownRelation verifies that inserting an
// application endpoint with an unknown relation fails.
//
// Ensures that an error is returned
func (s *applicationEndpointStateSuite) TestInsertApplicationEndpointUnknownRelation(c *gc.C) {
	// Arrange:
	// - One expected relation
	// - bind an unexpected relation
	db, err := s.state.DB()
	c.Assert(err, jc.ErrorIsNil)
	s.addRelation(c, "default")
	bindings := map[string]network.SpaceName{
		"unknown": "alpha",
	}

	// Act
	err = db.Txn(context.Background(), func(ctx context.Context, tx *sqlair.TX) error {
		return s.state.insertApplicationEndpoints(context.Background(), tx, insertApplicationEndpointsParams{
			appID:     s.appID,
			charmUUID: s.charmUUID,
			bindings:  bindings,
		})
	})

	// Assert: should fail because unknown is not a valid relation
	c.Assert(err, jc.ErrorIs, applicationerrors.CharmRelationNotFound)
}

// applicationEndpoint represents an association between a charm relation and a
// specific network space. It is used to fetch the state in order to verify what
// has been created
type applicationEndpoint struct {
	charmRelationUUID string
	spaceName         string
}

// fetchApplicationEndpoints retrieves a list of application endpoints from
// the database based on the application UUID.
//
// Returns a slice of applicationEndpoint containing charmRelationUUID and
// spaceName for each endpoint.
func (s *applicationEndpointStateSuite) fetchApplicationEndpoints(c *gc.C) []applicationEndpoint {
	nilEmpty := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	var endpoints []applicationEndpoint
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		rows, err := tx.Query(`
SELECT ae.charm_relation_uuid, s.name
FROM application_endpoint ae
LEFT JOIN space s ON s.uuid=ae.space_uuid
WHERE ae.application_uuid=?
ORDER BY s.name`, s.appID)
		defer func() { _ = rows.Close() }()
		if err != nil {
			return errors.Capture(err)
		}
		for rows.Next() {
			var uuid string
			var name *string
			if err := rows.Scan(&uuid, &name); err != nil {
				return errors.Capture(err)
			}
			endpoints = append(endpoints, applicationEndpoint{
				charmRelationUUID: uuid,
				spaceName:         nilEmpty(name),
			})
		}
		return nil
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Assert) Failed to fetch endpoints: %v", err))
	return endpoints
}

func (s *applicationEndpointStateSuite) fetchApplicationExtraEndpoints(c *gc.C) []applicationEndpoint {
	var endpoints []applicationEndpoint
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		rows, err := tx.Query(`
SELECT ae.charm_extra_binding_uuid, s.name
FROM application_extra_endpoint ae
LEFT JOIN space s ON s.uuid=ae.space_uuid
WHERE ae.application_uuid=?
ORDER BY s.name`, s.appID)
		defer func() { _ = rows.Close() }()
		if err != nil {
			return errors.Capture(err)
		}
		for rows.Next() {
			var uuid string
			var name *string
			if err := rows.Scan(&uuid, &name); err != nil {
				return errors.Capture(err)
			}
			endpoints = append(endpoints, applicationEndpoint{
				charmRelationUUID: uuid,
				spaceName:         deptr(name),
			})
		}
		return rows.Err()
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Assert) Failed to fetch extra endpoints: %v", err))
	return endpoints
}

// addSpace ensures a space with the given name exists in the database,
// creating it if necessary, and returns its name.
func (s *applicationEndpointStateSuite) addSpace(c *gc.C, name string) network.SpaceName {
	spaceUUID := uuid.MustNewUUID().String()
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO space (uuid, name)
VALUES (?, ?)`, spaceUUID, name)
		return errors.Capture(err)
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Arrange) Failed to add a space: %v", err))
	return network.SpaceName(name)
}

// addRelation inserts a new charm relation into the database and returns its generated UUID.
// It asserts that the operation succeeds and fails the test if an error occurs.
func (s *applicationEndpointStateSuite) addRelation(c *gc.C, name string) string {
	relUUID := uuid.MustNewUUID().String()
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO charm_relation (uuid, charm_uuid, kind_id, scope_id, role_id, name)
VALUES (?,?,0,0,0,?)`, relUUID, s.charmUUID, name)
		return errors.Capture(err)
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Arrange) Failed to add charm relation: %v", err))
	return relUUID
}

// addExtraBinding adds a new extra binding to the charm_extra_binding table
// and returns its generated UUID.
// It asserts that the operation succeeds and fails the test if an error occurs.
func (s *applicationEndpointStateSuite) addExtraBinding(c *gc.C, name string) string {
	bindingUUID := uuid.MustNewUUID().String()
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO charm_extra_binding (uuid, charm_uuid, name) 
VALUES (?,?,?)`, bindingUUID, s.charmUUID, name)
		return errors.Capture(err)
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Arrange) Failed to add charm extra binding: %v", err))
	return bindingUUID
}

func (s *applicationEndpointStateSuite) getApplicationDefaultSpace(c *gc.C) string {
	var spaceName string
	err := s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRow(`
SELECT s.name
FROM application a
JOIN space s ON s.uuid=a.space_uuid
WHERE a.uuid=?`, s.appID).Scan(&spaceName)
	})
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("(Assert) Failed to fetch default space: %v", err))
	return spaceName
}

// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package fortress_test

import (
	"context"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/worker/v4"
	gc "gopkg.in/check.v1"

	coretesting "github.com/juju/juju/internal/testing"
	"github.com/juju/juju/internal/worker/fortress"
)

type FortressSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&FortressSuite{})

func (s *FortressSuite) TestOutputBadSource(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	var dummy struct{ worker.Worker }
	var out fortress.Guard
	err := fix.manifold.Output(dummy, &out)
	c.Check(err, gc.ErrorMatches, "in should be \\*fortress\\.fortress; is .*")
	c.Check(out, gc.IsNil)
}

func (s *FortressSuite) TestOutputBadTarget(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	var out interface{}
	err := fix.manifold.Output(fix.worker, &out)
	c.Check(err.Error(), gc.Equals, "out should be *fortress.Guest or *fortress.Guard; is *interface {}")
	c.Check(out, gc.IsNil)
}

func (s *FortressSuite) TestStoppedUnlock(c *gc.C) {
	fix := newFixture(c)
	fix.TearDown(c)

	err := fix.Guard(c).Unlock(context.Background())
	c.Check(err, gc.Equals, fortress.ErrShutdown)
}

func (s *FortressSuite) TestStoppedLockdown(c *gc.C) {
	fix := newFixture(c)
	fix.TearDown(c)

	err := fix.Guard(c).Lockdown(context.Background())
	c.Check(err, gc.Equals, fortress.ErrShutdown)
}

func (s *FortressSuite) TestStoppedVisit(c *gc.C) {
	fix := newFixture(c)
	fix.TearDown(c)

	err := fix.Guest(c).Visit(context.Background(), nil)
	c.Check(err, gc.Equals, fortress.ErrShutdown)
}

func (s *FortressSuite) TestStartsLocked(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	AssertLocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestInitialLockdown(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	err := fix.Guard(c).Lockdown(context.Background())
	c.Check(err, jc.ErrorIsNil)
	AssertLocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestInitialUnlock(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	err := fix.Guard(c).Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)
	AssertUnlocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestDoubleUnlock(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	guard := fix.Guard(c)
	err := guard.Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)

	err = guard.Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)
	AssertUnlocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestDoubleLockdown(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	guard := fix.Guard(c)
	err := guard.Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)
	err = guard.Lockdown(context.Background())
	c.Check(err, jc.ErrorIsNil)

	err = guard.Lockdown(context.Background())
	c.Check(err, jc.ErrorIsNil)
	AssertLocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestWorkersIndependent(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	// Create a separate worker and associated guard from the same manifold.
	worker2, err := fix.manifold.Start(context.Background(), nil)
	c.Assert(err, jc.ErrorIsNil)
	defer CheckStop(c, worker2)
	var guard2 fortress.Guard
	err = fix.manifold.Output(worker2, &guard2)
	c.Assert(err, jc.ErrorIsNil)

	// Unlock the separate worker; check the original worker is unaffected.
	err = guard2.Unlock(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	AssertLocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestVisitError(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)
	err := fix.Guard(c).Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)

	err = fix.Guest(c).Visit(context.Background(), badVisit)
	c.Check(err, gc.ErrorMatches, "bad!")
}

func (s *FortressSuite) TestVisitSuccess(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)
	err := fix.Guard(c).Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)

	err = fix.Guest(c).Visit(context.Background(), func() error { return nil })
	c.Check(err, jc.ErrorIsNil)
}

func (s *FortressSuite) TestConcurrentVisit(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)
	err := fix.Guard(c).Unlock(context.Background())
	c.Check(err, jc.ErrorIsNil)
	guest := fix.Guest(c)

	// Start a bunch of concurrent, blocking, Visits.
	const count = 10
	var started sync.WaitGroup
	finishes := make(chan int, count)
	unblocked := make(chan struct{})
	for i := 0; i < count; i++ {
		started.Add(1)
		go func(i int) {
			visit := func() error {
				started.Done()
				<-unblocked
				return nil
			}
			err := guest.Visit(context.Background(), visit)
			c.Check(err, jc.ErrorIsNil)
			finishes <- i

		}(i)
	}
	started.Wait()

	// Just for fun, make sure a separate Visit still works as expected.
	AssertUnlocked(c, guest)

	// Unblock them all, and wait for them all to complete.
	close(unblocked)
	timeout := time.After(coretesting.LongWait)
	seen := make(map[int]bool)
	for i := 0; i < count; i++ {
		select {
		case finished := <-finishes:
			c.Logf("visit %d finished", finished)
			seen[finished] = true
		case <-timeout:
			c.Errorf("timed out waiting for %dth result", i)
		}
	}
	c.Check(seen, gc.HasLen, count)
}

func (s *FortressSuite) TestUnlockUnblocksVisit(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	// Start a Visit on a locked fortress, and check it's blocked.
	visited := make(chan error, 1)
	go func() {
		visited <- fix.Guest(c).Visit(context.Background(), badVisit)
	}()
	select {
	case err := <-visited:
		c.Fatalf("unexpected Visit result: %v", err)
	case <-time.After(coretesting.ShortWait):
	}

	// Unlock the fortress, and check the Visit is unblocked.
	err := fix.Guard(c).Unlock(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	select {
	case err := <-visited:
		c.Check(err, gc.ErrorMatches, "bad!")
	case <-time.After(coretesting.LongWait):
		c.Fatalf("timed out")
	}
}

func (s *FortressSuite) TestVisitUnblocksLockdown(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	// Start a long Visit to an unlocked fortress.
	unblockVisit := fix.startBlockingVisit(c)
	defer close(unblockVisit)

	// Start a Lockdown call, and check that nothing progresses...
	locked := make(chan error, 1)
	go func() {
		locked <- fix.Guard(c).Lockdown(context.Background())
	}()
	select {
	case err := <-locked:
		c.Fatalf("unexpected Lockdown result: %v", err)
	case <-time.After(coretesting.ShortWait):
	}

	// ...including new Visits.
	AssertLocked(c, fix.Guest(c))

	// Complete the running Visit, and check that the Lockdown completes too.
	unblockVisit <- struct{}{}
	select {
	case err := <-locked:
		c.Check(err, jc.ErrorIsNil)
	case <-time.After(coretesting.LongWait):
		c.Fatalf("timed out")
	}
}

func (s *FortressSuite) TestAbortedLockdownStillLocks(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	// Start a long Visit to an unlocked fortress.
	unblockVisit := fix.startBlockingVisit(c)
	defer close(unblockVisit)

	// Start a Lockdown call, and check that nothing progresses...
	locked := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		locked <- fix.Guard(c).Lockdown(ctx)
	}()
	select {
	case err := <-locked:
		c.Fatalf("unexpected Lockdown result: %v", err)
	case <-time.After(coretesting.ShortWait):
	}

	// ...then abort the lockdown.
	cancel()
	select {
	case err := <-locked:
		c.Check(err, gc.Equals, fortress.ErrAborted)
	case <-time.After(coretesting.LongWait):
		c.Fatalf("timed out")
	}

	// Check the fortress is already locked, even as the old visit continues.
	AssertLocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestAbortedLockdownUnlock(c *gc.C) {
	fix := newFixture(c)
	defer fix.TearDown(c)

	// Start a long Visit to an unlocked fortress.
	unblockVisit := fix.startBlockingVisit(c)
	defer close(unblockVisit)

	// Start and abort a Lockdown.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	guard := fix.Guard(c)
	err := guard.Lockdown(ctx)
	c.Assert(err, gc.Equals, fortress.ErrAborted)

	// Unlock the fortress again, leaving the original visit running, and
	// check that new Visits are immediately accepted.
	err = guard.Unlock(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	AssertUnlocked(c, fix.Guest(c))
}

func (s *FortressSuite) TestIsFortressError(c *gc.C) {
	c.Check(fortress.IsFortressError(fortress.ErrAborted), jc.IsTrue)
	c.Check(fortress.IsFortressError(fortress.ErrShutdown), jc.IsTrue)
	c.Check(fortress.IsFortressError(errors.Trace(fortress.ErrShutdown)), jc.IsTrue)
	c.Check(fortress.IsFortressError(errors.New("boom")), jc.IsFalse)
}

// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// leaderGate flips the collector on for as long as this replica holds
// leadership. When leader election is disabled the manager starts leader
// election runnables immediately, so single replica installs report normally.
type leaderGate struct {
	collector *Collector
}

var (
	_ manager.Runnable               = &leaderGate{}
	_ manager.LeaderElectionRunnable = &leaderGate{}
)

// LeaderRunnable returns a Runnable that must be added to the manager for the
// collector to report anything.
func (c *Collector) LeaderRunnable() manager.Runnable {
	return &leaderGate{collector: c}
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
func (g *leaderGate) NeedLeaderElection() bool {
	return true
}

// Start implements manager.Runnable. The manager syncs the caches it knows
// about before starting leader election runnables, but the Order and
// ArtifactWorkflow informers are created lazily by the controllers, which are
// leader election runnables started alongside this one. A scrape that arrives
// before a controller has asked for its informer therefore creates it and waits
// for it to sync, bounded by collectTimeout, so the worst case is a scrape that
// fails rather than one that hangs.
func (g *leaderGate) Start(ctx context.Context) error {
	g.collector.isLeader.Store(true)
	defer g.collector.isLeader.Store(false)

	<-ctx.Done()

	return nil
}

// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	_ manager.Runnable               = &Collector{}
	_ manager.LeaderElectionRunnable = &Collector{}
)

// NeedLeaderElection implements manager.LeaderElectionRunnable. Every replica
// serves the metrics endpoint but only the leader reconciles, so ungated gauges
// would be multiplied by the replica count.
func (c *Collector) NeedLeaderElection() bool {
	return true
}

// Start implements manager.Runnable, and reports for as long as this replica
// holds leadership. The collector must be added to the manager or it reports
// nothing at all. When leader election is disabled the manager starts leader
// election runnables immediately, so single replica installs report normally.
//
// The manager syncs the caches it knows about before starting leader election
// runnables, but the Order and ArtifactWorkflow informers are created lazily by
// the controllers, which are leader election runnables started alongside this
// one. A scrape that arrives before a controller has asked for its informer
// therefore creates it and waits for it to sync, bounded by collectTimeout, so
// the worst case is a scrape that fails rather than one that hangs.
func (c *Collector) Start(ctx context.Context) error {
	c.isLeader.Store(true)
	defer c.isLeader.Store(false)

	<-ctx.Done()

	return nil
}

// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	"github.com/robfig/cron/v3"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

// ScheduleInterval returns the gap a cron ArtifactWorkflow declares between
// runs, taken as the shortest gap across its schedules. The gap is measured
// between the next two activations rather than assumed, so @every, @daily and a
// plain five field expression all resolve the same way.
//
// An interval is what makes elapsed time judgeable: two hours since the last
// success is late for a two minute schedule and early for a daily one. The
// second return value is false when nothing usable is declared, in which case
// no series is reported rather than a made up number.
func ScheduleInterval(spec *arcv1alpha1.Cron, now time.Time) (int64, bool) {
	if spec == nil {
		return 0, false
	}

	if spec.Timezone != "" {
		if loc, err := time.LoadLocation(spec.Timezone); err == nil {
			now = now.In(loc)
		}
	}

	var shortest int64

	for _, schedule := range spec.Schedules {
		parsed, err := cron.ParseStandard(schedule)
		if err != nil {
			continue
		}

		// Two activations ahead of now, so the gap is the one that is coming
		// rather than one inferred from a run that may have been missed.
		first := parsed.Next(now)
		if first.IsZero() {
			continue
		}

		second := parsed.Next(first)
		if second.IsZero() {
			continue
		}

		gap := int64(second.Sub(first).Seconds())
		if gap <= 0 {
			continue
		}

		if shortest == 0 || gap < shortest {
			shortest = gap
		}
	}

	return shortest, shortest > 0
}

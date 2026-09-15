// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	"github.com/robfig/cron/v3"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

// ScheduleInterval returns how long a cron ArtifactWorkflow was meant to wait
// after the given time before running again, taken to the soonest activation
// across its schedules. Parsing the schedule rather than assuming a period means
// @every, @daily and a plain five field expression all resolve the same way.
//
// The anchor has to be the run being judged, not the current time. Gaps are not
// always equal: "0 9,17 * * *" alternates between eight and sixteen hours, so a
// gap read at scrape time belongs to whichever pair of activations happens to
// come next and says nothing about the run that actually finished.
//
// An interval is what makes elapsed time judgeable: two hours since the last
// success is late for a two minute schedule and early for a daily one.
// Subtracting it from that same elapsed time gives the amount by which the next
// run is overdue, for any schedule, regular or not.
//
// The second return value is false when nothing usable is declared, in which
// case no series is reported rather than a made up number.
func ScheduleInterval(spec *arcv1alpha1.Cron, after time.Time) (int64, bool) {
	if spec == nil {
		return 0, false
	}

	if spec.Timezone != "" {
		if loc, err := time.LoadLocation(spec.Timezone); err == nil {
			after = after.In(loc)
		}
	}

	// Several schedules mean several candidates; the run is due at the first of
	// them, so the soonest wins rather than the shortest declared period.
	var next time.Time

	for _, schedule := range spec.Schedules {
		parsed, err := cron.ParseStandard(schedule)
		if err != nil {
			continue
		}

		candidate := parsed.Next(after)
		if candidate.IsZero() {
			continue
		}

		if next.IsZero() || candidate.Before(next) {
			next = candidate
		}
	}

	if next.IsZero() {
		return 0, false
	}

	gap := int64(next.Sub(after).Seconds())
	if gap <= 0 {
		return 0, false
	}

	return gap, true
}

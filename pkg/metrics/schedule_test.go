// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScheduleInterval", func() {
	// A Monday, so weekday only expressions resolve without a weekend gap.
	monday := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	at := func(hour, minute int) time.Time {
		return monday.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
	}

	It("should read the gap from every schedule form the CRD allows", func() {
		for _, tc := range []struct {
			schedule string
			want     int64
		}{
			{"*/2 * * * *", 120},
			{"*/15 * * * *", 900},
			{"0 * * * *", 3600},
			{"@hourly", 3600},
			{"@daily", 86400},
			{"@every 90s", 90},
			{"@every 6h", 21600},
		} {
			// Anchored on an activation, which is where a real success lands.
			got, ok := ScheduleInterval(&arcv1alpha1.Cron{Schedules: []string{tc.schedule}}, monday)
			Expect(ok).To(BeTrue(), "schedule %q should resolve", tc.schedule)
			Expect(got).To(Equal(tc.want), "schedule %q", tc.schedule)
		}
	})

	// The gaps of an uneven schedule are not interchangeable. Reading one at
	// scrape time reports whichever pair comes next, which is not the gap that
	// followed the run being judged.
	It("should follow the anchor when the gaps are uneven", func() {
		twiceDaily := &arcv1alpha1.Cron{Schedules: []string{"0 9,17 * * *"}}

		morning, ok := ScheduleInterval(twiceDaily, at(9, 0))
		Expect(ok).To(BeTrue())
		Expect(morning).To(Equal(int64(8*3600)), "09:00 is followed by 17:00, eight hours later")

		evening, ok := ScheduleInterval(twiceDaily, at(17, 0))
		Expect(ok).To(BeTrue())
		Expect(evening).To(Equal(int64(16*3600)), "17:00 is followed by 09:00 the next day")
	})

	It("should take the soonest activation when several schedules are declared", func() {
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Schedules: []string{"@daily", "*/5 * * * *", "@hourly"},
		}, monday)

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(300)), "the five minute schedule comes round first")
	})

	It("should measure from the anchor rather than from the schedule", func() {
		// A success lands when the run finished, which is a little after the
		// activation, so the remaining wait is shorter than the full period.
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Schedules: []string{"*/5 * * * *"},
		}, at(12, 2))

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(180)), "12:02 is three minutes short of the 12:05 activation")
	})

	It("should report nothing rather than invent a number", func() {
		for _, spec := range []*arcv1alpha1.Cron{
			nil,
			{Schedules: nil},
			{Schedules: []string{"not a schedule"}},
		} {
			_, ok := ScheduleInterval(spec, monday)
			Expect(ok).To(BeFalse())
		}
	})

	It("should ignore an unparseable schedule beside a good one", func() {
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Schedules: []string{"nonsense", "@hourly"},
		}, monday)

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(3600)))
	})

	It("should resolve against the declared timezone", func() {
		midnight := []string{"0 0 * * *"}

		// The anchor is midnight UTC, which is 09:00 in Tokyo, so the next
		// Tokyo midnight is fifteen hours away rather than a full day.
		tokyo, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Timezone:  "Asia/Tokyo",
			Schedules: midnight,
		}, monday)

		Expect(ok).To(BeTrue())
		Expect(tokyo).To(Equal(int64(15*3600)))

		utc, ok := ScheduleInterval(&arcv1alpha1.Cron{Schedules: midnight}, monday)

		Expect(ok).To(BeTrue())
		Expect(utc).To(Equal(int64(86400)), "without a timezone the same anchor sits on the activation")
	})
})

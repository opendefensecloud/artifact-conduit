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
	now := time.Date(2026, 3, 2, 10, 17, 0, 0, time.UTC)

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
			got, ok := ScheduleInterval(&arcv1alpha1.Cron{Schedules: []string{tc.schedule}}, now)
			Expect(ok).To(BeTrue(), "schedule %q should resolve", tc.schedule)
			Expect(got).To(Equal(tc.want), "schedule %q", tc.schedule)
		}
	})

	It("should take the shortest gap when several schedules are declared", func() {
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Schedules: []string{"@daily", "*/5 * * * *", "@hourly"},
		}, now)

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(300)), "the five minute schedule is the one that comes round first")
	})

	It("should report nothing rather than invent a number", func() {
		for _, spec := range []*arcv1alpha1.Cron{
			nil,
			{Schedules: nil},
			{Schedules: []string{"not a schedule"}},
		} {
			_, ok := ScheduleInterval(spec, now)
			Expect(ok).To(BeFalse())
		}
	})

	It("should ignore an unparseable schedule beside a good one", func() {
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Schedules: []string{"nonsense", "@hourly"},
		}, now)

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(3600)))
	})

	It("should resolve against the declared timezone", func() {
		got, ok := ScheduleInterval(&arcv1alpha1.Cron{
			Timezone:  "Asia/Tokyo",
			Schedules: []string{"@daily"},
		}, now)

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(int64(86400)))
	})
})

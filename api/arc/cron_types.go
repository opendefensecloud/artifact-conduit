// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

// Cron represents an order's cron schedule.
type Cron struct {
	// Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time.
	Timezone string `json:"timezone,omitempty"`
	// StartingDeadlineSeconds is the K8s-style deadline that will limit the time a Order will be run after its
	// original scheduled time if it is missed.
	// +kubebuilder:validation:Minimum=0
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`
	// Schedules is a list of schedules to run the Order in Cron format
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Pattern=`^(@(yearly|annually|monthly|weekly|daily|midnight|hourly)|@every\s+([0-9]+(ns|us|µs|ms|s|m|h))+|([0-9*,/?-]+\s+){4}[0-9*,/?-]+)$`
	Schedules []string `json:"schedules"`
	// When is an expression that determines if a run should be scheduled.
	When string `json:"when,omitempty"`
}

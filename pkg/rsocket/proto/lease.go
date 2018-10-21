package proto

import "time"

// LeaseOptions configures the LEASE frame
type LeaseOption struct {
	TimeToLive time.Duration // Time for validity of LEASE from time of reception.
	Requests   uint          // Number of Requests that may be sent until next LEASE.
}

func NewLeaseOption() *LeaseOption {
	return &LeaseOption{0, 0}
}

func (lease *LeaseOption) Enabled() bool {
	return lease.TimeToLive > 0 && lease.Requests > 0
}

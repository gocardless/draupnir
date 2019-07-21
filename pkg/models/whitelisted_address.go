package models

import (
	"time"
)

type WhitelistedAddress struct {
	// Given that we're not serving this model via JSON:API, we don't need a
	// surrogate key (e.g. 'ID'). The IP address and instance ID are used as a composite key.
	IPAddress string
	Instance  *Instance
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewWhitelistedAddress(ipaddress string, instance *Instance) WhitelistedAddress {
	return WhitelistedAddress{
		IPAddress: ipaddress,
		Instance:  instance,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

package dashboard

import (
	"time"

	"github.com/google/uuid"
)

// SliceType represents different types of network slices
type SliceType string

const (
	SliceTypeEmbb   SliceType = "eMBB"
	SliceTypeUrllc  SliceType = "URLLC"
	SliceTypeMassIoT SliceType = "mMTC"
)

// SliceState represents the current state of a network slice
type SliceState string

const (
	SliceStateCreated     SliceState = "CREATED"
	SliceStateActive      SliceState = "ACTIVE"
	SliceStateInactive    SliceState = "INACTIVE"
	SliceStateScaling     SliceState = "SCALING"
	SliceStateFailed      SliceState = "FAILED"
	SliceStateTerminating SliceState = "TERMINATING"
)

// NetworkSlice represents a complete network slice configuration
type NetworkSlice struct {
	ID             uuid.UUID        `json:"id"`
	Name           string           `json:"name"`
	Type           SliceType        `json:"type"`
	State          SliceState       `json:"state"`
	Tenant         string           `json:"tenant"`
	ServiceProfile ServiceProfile   `json:"serviceProfile"`
	ResourceQuota  ResourceQuota    `json:"resourceQuota"`
	SLA            SliceSLA         `json:"sla"`
	Placement      PlacementPolicy  `json:"placement"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

// ServiceProfile defines the performance characteristics of a slice
type ServiceProfile struct {
	MaxDataRate struct {
		Downlink int `json:"downlink"` // Mbps
		Uplink   int `json:"uplink"`   // Mbps
	} `json:"maxDataRate"`
	Latency          int     `json:"latency"`          // ms
	Reliability      float64 `json:"reliability"`       // percentage
	MaxConcurrentUEs int     `json:"maxConcurrentUEs"`
}

// ResourceQuota defines compute, network, and storage resources
type ResourceQuota struct {
	Compute struct {
		CPU struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"cpu"`
		Memory struct {
			Min int `json:"min"` // GB
			Max int `json:"max"`
		} `json:"memory"`
	} `json:"compute"`
	Network struct {
		Bandwidth struct {
			Min        int `json:"min"`
			Guaranteed int `json:"guaranteed"`
			Max        int `json:"max"`
		} `json:"bandwidth"`
		VLANs []int `json:"vlans"`
	} `json:"network"`
	Storage struct {
		Capacity int `json:"capacity"` // GB
		IOPS     int `json:"iops"`
	} `json:"storage"`
}

// SliceSLA defines service level agreements and monitoring
type SliceSLA struct {
	Objectives struct {
		Throughput struct {
			Downlink struct {
				Guaranteed int `json:"guaranteed"`
				Maximum    int `json:"maximum"`
			} `json:"downlink"`
			Uplink struct {
				Guaranteed int `json:"guaranteed"`
				Maximum    int `json:"maximum"`
			} `json:"uplink"`
		} `json:"throughput"`
		Latency struct {
			E2E struct {
				Target  int `json:"target"`
				Maximum int `json:"maximum"`
			} `json:"e2e"`
			RAN struct {
				Target  int `json:"target"`
				Maximum int `json:"maximum"`
			} `json:"ran"`
		} `json:"latency"`
		Reliability float64 `json:"reliability"`
		PacketLoss  float64 `json:"packetLoss"`
	} `json:"objectives"`
	
	Monitoring struct {
		Interval int      `json:"interval"`
		Metrics  []string `json:"metrics"`
	} `json:"monitoring"`
}

// PlacementPolicy defines rules for slice deployment
type PlacementPolicy struct {
	Regions []struct {
		Name  string   `json:"name"`
		Zones []string `json:"zones"`
	} `json:"regions"`
	Affinity []struct {
		Key      string   `json:"key"`
		Values   []string `json:"values"`
		Operator string   `json:"operator"`
	} `json:"affinity"`
	AntiAffinity []struct {
		Key      string   `json:"key"`
		Values   []string `json:"values"`
		Operator string   `json:"operator"`
	} `json:"antiAffinity"`
}

// SliceEvent represents important lifecycle events
type SliceEvent struct {
	Type        string    `json:"type"`
	SliceID     uuid.UUID `json:"sliceId"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}
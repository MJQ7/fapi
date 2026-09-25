package core

import (
	"sync"
	"time"
)

// The dashboard's charts count requests in 5-second slots over the last
// hour. The counts are kept in memory, whether or not the request log is on,
// and start again when fapi restarts.
const (
	trafficSlot    = 5 * time.Second
	TrafficHistory = time.Hour // how far back Traffic can look
	trafficSlots   = int(TrafficHistory / trafficSlot)
)

// TrafficCounts is how many requests a port received, split by what answered
// them. Received is always Proxied plus Sent.
type TrafficCounts struct {
	Received int `json:"received"`
	Proxied  int `json:"proxied"` // forwarded to the real API
	Sent     int `json:"sent"`    // answered by fapi itself: an endpoint or a CORS preflight
}

// TrafficReport is the traffic on each port over a period, in equal steps,
// oldest first.
type TrafficReport struct {
	Start       time.Time               `json:"start"` // when the first step began
	StepSeconds int                     `json:"stepSeconds"`
	Ports       map[int][]TrafficCounts `json:"ports"` // only the ports that received requests
}

// trafficCounter keeps the counts in a ring of slots, reusing the oldest slot
// as time moves on.
type trafficCounter struct {
	mutex sync.Mutex
	slots [trafficSlots]trafficSlotCounts
}

type trafficSlotCounts struct {
	number int64 // the slot's start, counted in slots since the Unix epoch
	ports  map[int]*TrafficCounts
}

// add counts one request that arrived on port at the time given.
func (t *trafficCounter) add(at time.Time, port int, proxied bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	number := slotNumber(at)
	slot := &t.slots[number%int64(trafficSlots)]
	if slot.number != number || slot.ports == nil {
		*slot = trafficSlotCounts{number: number, ports: map[int]*TrafficCounts{}}
	}

	counts := slot.ports[port]
	if counts == nil {
		counts = &TrafficCounts{}
		slot.ports[port] = counts
	}
	counts.Received++
	if proxied {
		counts.Proxied++
	} else {
		counts.Sent++
	}
}

// report adds up the slots in period, ending now, in steps of step. period
// and step must be whole numbers of slots, and period a whole number of
// steps. Steps line up with the clock (a 1-minute step starts on the
// minute), so a chart's history doesn't shift as it updates.
func (t *trafficCounter) report(now time.Time, period time.Duration, step time.Duration) TrafficReport {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	steps := int(period / step)
	slotsPerStep := int64(step / trafficSlot)
	end := (slotNumber(now)/slotsPerStep + 1) * slotsPerStep // the slot after the current step
	start := end - int64(steps)*slotsPerStep

	report := TrafficReport{
		Start:       time.Unix(start*int64(trafficSlot/time.Second), 0).UTC(),
		StepSeconds: int(step / time.Second),
		Ports:       map[int][]TrafficCounts{},
	}
	for _, slot := range t.slots {
		if slot.number < start || slot.number >= end {
			continue
		}
		index := (slot.number - start) / slotsPerStep
		for port, counts := range slot.ports {
			if report.Ports[port] == nil {
				report.Ports[port] = make([]TrafficCounts, steps)
			}
			total := &report.Ports[port][index]
			total.Received += counts.Received
			total.Proxied += counts.Proxied
			total.Sent += counts.Sent
		}
	}
	return report
}

func slotNumber(at time.Time) int64 {
	return at.Unix() / int64(trafficSlot/time.Second)
}

// Traffic returns how many requests each port received over the last period,
// in steps of step. Both must be whole multiples of 5 seconds, period no
// more than TrafficHistory, and period a whole number of steps.
func (c *Core) Traffic(period time.Duration, step time.Duration) TrafficReport {
	return c.traffic.report(time.Now(), period, step)
}

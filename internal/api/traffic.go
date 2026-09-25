package api

import (
	"net/http"
	"time"
)

// trafficSteps is how many points each of the dashboard's charts has.
const trafficSteps = 60

// trafficPeriods are the periods GET /api/traffic can report on, in minutes.
// Each is split into trafficSteps steps, which must be a whole number of the
// counter's 5-second slots.
var trafficPeriods = map[string]time.Duration{
	"5":  5 * time.Minute,
	"15": 15 * time.Minute,
	"30": 30 * time.Minute,
	"60": 60 * time.Minute,
}

// getTraffic answers GET /api/traffic?minutes=15: how many requests each
// port received over the last 5, 15, 30 or 60 minutes (15 when not given),
// in 60 steps, for the dashboard's charts.
func (a *API) getTraffic(writer http.ResponseWriter, request *http.Request) {
	minutes := request.URL.Query().Get("minutes")
	if minutes == "" {
		minutes = "15"
	}
	period, ok := trafficPeriods[minutes]
	if !ok {
		writeError(writer, http.StatusBadRequest, "minutes must be 5, 15, 30 or 60")
		return
	}
	writeJSON(writer, http.StatusOK, a.core.Traffic(period, period/trafficSteps))
}

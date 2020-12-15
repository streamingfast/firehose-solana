package metrics

import (
	"time"

	"github.com/dfuse-io/dmetrics"
)

var headBlockTime time.Time

var Metricset = dmetrics.NewSet()

var HeadBlockTimeDrift = Metricset.NewHeadTimeDrift("serumdb-loader")
var HeadBlockNumber = Metricset.NewHeadBlockNumber("serumdb-loader")

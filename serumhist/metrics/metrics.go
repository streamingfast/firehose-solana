package metrics

import (
	"time"

	"github.com/dfuse-io/dmetrics"
)

var headBlockTime time.Time

var Metricset = dmetrics.NewSet()

var HeadBlockTimeDrift = Metricset.NewHeadTimeDrift("serumhist-loader")
var HeadBlockNumber = Metricset.NewHeadBlockNumber("serumhist-loader")

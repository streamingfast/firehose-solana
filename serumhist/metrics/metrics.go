package metrics

import (
	"time"

	"github.com/dfuse-io/dmetrics"
)

var headBlockTime time.Time

var Metricset = dmetrics.NewSet()

var HeadBlockTimeDrift = Metricset.NewHeadTimeDrift("serumhist-loader")
var HeadBlockNumber = Metricset.NewHeadBlockNumber("serumhist-loader")

//var TradingAccountCount = Metricset.NewGauge("trading_account_count")
//var LastFlushedSlotNum = Metricset.NewGauge("last_flush_slot_num")
//var ActiveOrderTrackingSubscription = Metricset.NewGauge("active_order_tracking_subscription")

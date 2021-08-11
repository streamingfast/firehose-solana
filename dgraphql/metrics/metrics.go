// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/streamingfast/dmetrics"
)

var Metricset = dmetrics.NewSet()

var QueryResponseTimes = Metricset.NewHistogram("query_response_times", "query response times histogram for percentile sampling")
var InflightQueryCount = Metricset.NewGauge("inflight_query_count", "inflight query count currently active")
var InflightSubscriptionCount = Metricset.NewGauge("inflight_subscription_count", "inflight subscription count currently active")
var HeadBlockNum = Metricset.NewHeadBlockNumber("dgraphql")
var HeadTimeDrift = Metricset.NewHeadTimeDrift("dgraphql")

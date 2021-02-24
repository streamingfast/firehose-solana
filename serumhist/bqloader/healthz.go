package bqloader

func (bq *BQLoader) Healthy() bool {
	return bq.loader.IsHealthy()
}

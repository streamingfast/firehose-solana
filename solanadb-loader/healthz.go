package solanadb_loader

func (l *Loader) setUnhealthy() {
	if l.healthy {
		l.healthy = false
	}
}

func (l *Loader) setHealthy() {
	if !l.healthy {
		l.healthy = true
	}
}

func (l *Loader) Healthy() bool {
	return l.healthy
}

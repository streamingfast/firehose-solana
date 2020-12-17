package serumhist

import "github.com/dfuse-io/dfuse-solana/serumhist/loader"

func (l *loader.Injector) setUnhealthy() {
	if l.healthy {
		l.healthy = false
	}
}

func (l *loader.Injector) setHealthy() {
	if !l.healthy {
		l.healthy = true
	}
}

func (l *loader.Injector) Healthy() bool {
	return l.healthy
}

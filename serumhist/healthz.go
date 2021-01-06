package serumhist

import (
	"net/http"

	"go.uber.org/zap"
)

func (i *Injector) LaunchHealthz(httpListenAddr string) {
	httpSrv := &http.Server{
		Addr:    httpListenAddr,
		Handler: http.HandlerFunc(i.healthz),
	}
	zlog.Info("starting serum histoy injector http health server",
		zap.String("http_addr", httpListenAddr),
	)

	go httpSrv.ListenAndServe()
}

func (i *Injector) healthz(w http.ResponseWriter, r *http.Request) {
	if !i.Healthy() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	w.Write([]byte("ready\n"))
}

func (i *Injector) SetUnhealthy() {
	if i.healthy {
		i.healthy = false
	}
}

func (i *Injector) setHealthy() {
	if !i.healthy {
		i.healthy = true
	}
}

func (i *Injector) Healthy() bool {
	return i.healthy
}

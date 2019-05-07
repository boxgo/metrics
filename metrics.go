package metrics

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/boxgo/box/minibox"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

type (
	// Metrics config
	Metrics struct {
		Namespace     string `config:"namespace"`
		Subsystem     string `config:"subsystem"`
		PushEnabled   bool   `config:"pushEnabled"`
		PushTargetURL string `config:"pushTargetURL"`
		PushJobName   string `config:"pushJobName" desc:"default is AppName"`
		PushInterval  int    `config:"pushInterval" desc:"seconds, default is 3"`

		name string
		app  minibox.App
		stop chan bool
	}
)

var (
	// Default metrics
	Default = New("metrics")
)

// Name config name
func (m *Metrics) Name() string {
	return m.name
}

// Exts app
func (m *Metrics) Exts() []minibox.MiniBox {
	return []minibox.MiniBox{&m.app}
}

// ConfigWillLoad before load
func (m *Metrics) ConfigWillLoad(context.Context) {

}

// ConfigDidLoad after load
func (m *Metrics) ConfigDidLoad(context.Context) {
	if m.PushJobName == "" {
		m.PushJobName = m.app.AppName
	}

	if m.PushInterval <= 0 {
		m.PushInterval = 3
	}

	if m.PushEnabled && (m.PushTargetURL == "" || m.PushJobName == "") {
		panic("config invalid: pushTargetURL, pushJobName must be set when pushEnabled is true")
	}
}

// Serve start serve
func (m *Metrics) Serve(context.Context) error {
	if !m.PushEnabled {
		return nil
	}

	hostname, _ := os.Hostname()

	go func() {
		ticker := time.NewTicker(time.Duration(m.PushInterval) * time.Second)
		defer ticker.Stop()

		pusher := push.
			New(m.PushTargetURL, m.PushJobName).
			Gatherer(prometheus.DefaultRegisterer.(prometheus.Gatherer)).
			Grouping("instance", hostname)

		for {
			select {
			case <-m.stop:
				break
			case <-ticker.C:
				pusher.Add()
			}
		}
	}()

	return nil
}

// Shutdown close clients when Shutdown
func (m *Metrics) Shutdown(context.Context) error {
	if !m.PushEnabled {
		return nil
	}

	go func() {
		m.stop <- true
	}()

	return nil
}

// Metrics metrics http
func (m *Metrics) Metrics() http.Handler {
	return promhttp.Handler()
}

// New a metrics
func New(name string) *Metrics {
	return &Metrics{
		name: name,
	}
}

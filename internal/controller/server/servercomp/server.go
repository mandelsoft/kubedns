package servercomp

import (
	"context"
	"net/http"

	"github.com/mandelsoft/logging"
)

type WebServer struct {
	port   string
	logger logging.Logger
	server *http.Server
	mux    *http.ServeMux
}

func NewWebServer(port string, logger logging.Logger) *WebServer {
	return &WebServer{
		port:   port,
		logger: logger,
		mux:    http.NewServeMux(),
	}
}

func (ws *WebServer) AddEndpoint(p string, h func(http.ResponseWriter, *http.Request)) {
	ws.mux.HandleFunc(p, h)
}

func (ws *WebServer) Start(ctx context.Context) error {
	ws.mux.HandleFunc("/api/status", ws.statusHandler)
	ws.mux.HandleFunc("/api/metrics", ws.metricsHandler)

	ws.logger.Info("starting web server on port {{port}}", "port", ws.port)
	ws.server = &http.Server{
		Addr:    ws.port,
		Handler: ws.mux,
	}

	// Shutdown gracefully when context is cancelled
	go func() {
		<-ctx.Done()
		ws.logger.Info("shutting down webserver")
		ws.server.Shutdown(context.Background())
	}()

	if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (ws *WebServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "running"}`))
}

func (ws *WebServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("custom_metric 1\n"))
}

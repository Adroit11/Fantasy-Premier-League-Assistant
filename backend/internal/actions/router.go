package actions

import (
	"encoding/json"
	"net/http"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type Router struct {
	mux                    *http.ServeMux
	connectTeamAction      *ConnectTeamAction
	getTeamOverviewAction  *GetTeamOverviewAction
	getAvailabilityAction  *GetAvailabilityNewsAction
	calculateProjectionsAction *CalculateProjectionsAction
	suggestLineupAction    *SuggestLineupAction
	suggestTransfersAction *SuggestTransfersAction
}

func NewRouter(client fpl.Client, engine scoring.Engine, optimizer scoring.Optimizer) *Router {
	r := &Router{
		mux:                        http.NewServeMux(),
		connectTeamAction:          NewConnectTeamAction(client),
		getTeamOverviewAction:      NewGetTeamOverviewAction(client),
		getAvailabilityAction:      NewGetAvailabilityNewsAction(client),
		calculateProjectionsAction: NewCalculateProjectionsAction(client, engine),
		suggestLineupAction:        NewSuggestLineupAction(client, engine, optimizer),
		suggestTransfersAction:     NewSuggestTransfersAction(client, engine),
	}

	r.routes()
	return r
}

func (r *Router) routes() {
	// Health & Status
	r.mux.HandleFunc("GET /healthz", r.handleHealth)

	// Single Action Endpoints
	r.mux.HandleFunc("POST /api/v1/team/connect", r.connectTeamAction.ServeHTTP)
	r.mux.HandleFunc("POST /api/v1/team/overview", r.getTeamOverviewAction.ServeHTTP)
	r.mux.HandleFunc("POST /api/v1/availability/news", r.getAvailabilityAction.ServeHTTP)
	r.mux.HandleFunc("POST /api/v1/projections/calculate", r.calculateProjectionsAction.ServeHTTP)
	r.mux.HandleFunc("POST /api/v1/lineup/suggest", r.suggestLineupAction.ServeHTTP)
	r.mux.HandleFunc("POST /api/v1/transfers/suggest", r.suggestTransfersAction.ServeHTTP)
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "fpl-assistant-core",
		"version": "1.0.0",
		"season":  "2026/2027",
	})
}

// ServeHTTP satisfies the http.Handler interface with CORS headers
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	r.mux.ServeHTTP(w, req)
}

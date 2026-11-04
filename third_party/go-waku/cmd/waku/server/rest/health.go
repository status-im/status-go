package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/waku/v2/node"
)

type HealthService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

const routeHealth = "/health"

func NewHealthService(node *node.WakuNode, m *chi.Mux) *HealthService {
	h := &HealthService{
		node: node,
		mux:  m,
	}

	m.Get(routeHealth, h.getHealth)

	return h
}

type HealthResponse string

func (d *HealthService) getHealth(w http.ResponseWriter, r *http.Request) {
	if d.node.RLNRelay() != nil {
		isReady, err := d.node.RLNRelay().IsReady(r.Context())
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				writeResponse(w, HealthResponse("Health check timed out"), http.StatusInternalServerError)
			} else {
				writeResponse(w, HealthResponse(err.Error()), http.StatusInternalServerError)
			}
			return
		}

		if isReady {
			writeResponse(w, HealthResponse("Node is healthy"), http.StatusOK)
		} else {
			writeResponse(w, HealthResponse("Node is not ready"), http.StatusInternalServerError)
		}
	} else {
		writeResponse(w, HealthResponse("Non RLN healthcheck is not implemented"), http.StatusNotImplemented)
	}
}

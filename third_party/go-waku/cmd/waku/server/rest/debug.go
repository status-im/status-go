package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/waku/v2/node"
)

type DebugService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

type InfoArgs struct {
}

type InfoReply struct {
	ENRUri          string   `json:"enrUri,omitempty"`
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

const routeDebugInfoV1 = "/debug/v1/info"
const routeDebugVersionV1 = "/debug/v1/version"

func NewDebugService(node *node.WakuNode, m *chi.Mux) *DebugService {
	d := &DebugService{
		node: node,
		mux:  m,
	}

	m.Get(routeDebugInfoV1, d.getV1Info)
	m.Get(routeDebugVersionV1, d.getV1Version)

	return d
}

type VersionResponse string

func (d *DebugService) getV1Info(w http.ResponseWriter, req *http.Request) {
	response := new(InfoReply)
	response.ENRUri = d.node.ENR().String()
	for _, addr := range d.node.ListenAddresses() {
		response.ListenAddresses = append(response.ListenAddresses, addr.String())
	}
	writeErrOrResponse(w, nil, response)
}

func (d *DebugService) getV1Version(w http.ResponseWriter, req *http.Request) {
	response := VersionResponse(node.GetVersionInfo().String())
	writeErrOrResponse(w, nil, response)
}

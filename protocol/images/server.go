package images

import (
	"context"
	"database/sql"
	"net"
	"net/http"

	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/identity/identicon"
)

type messageHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

type identiconHandler struct {
	logger *zap.Logger
}

func (s *identiconHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pks, ok := r.URL.Query()["publicKey"]
	if !ok || len(pks) == 0 {
		s.logger.Error("no publicKey")
		return
	}
	pk := pks[0]
	image, err := identicon.Generate(pk)
	if err != nil {
		s.logger.Error("could not generate identicon")
	}

	w.Header().Set("Content-Type", "image/png")
	_, err = w.Write(image)
	if err != nil {
		s.logger.Error("failed to write image", zap.Error(err))
	}

}

func (s *messageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	messageIDs, ok := r.URL.Query()["messageId"]
	if !ok || len(messageIDs) == 0 {
		s.logger.Error("no messageID")
		return
	}
	messageID := messageIDs[0]
	var image []byte
	err := s.db.QueryRow(`SELECT image_payload FROM user_messages WHERE id = ?`, messageID).Scan(&image)
	if err != nil {
		s.logger.Error("failed to find image", zap.Error(err))
		return
	}
	if len(image) == 0 {
		s.logger.Error("empty image")
		return
	}
	mime, err := ImageMime(image)
	if err != nil {
		s.logger.Error("failed to get mime", zap.Error(err))
	}

	w.Header().Set("Content-Type", mime)
	_, err = w.Write(image)
	if err != nil {
		s.logger.Error("failed to write image", zap.Error(err))
	}
}

type Server struct {
	Port   int
	server *http.Server
	logger *zap.Logger
	db     *sql.DB
}

func NewServer(db *sql.DB, logger *zap.Logger) *Server {
	return &Server{db: db, logger: logger}

}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	s.Port = listener.Addr().(*net.TCPAddr).Port
	handler := http.NewServeMux()
	handler.Handle("/messages/images", &messageHandler{db: s.db, logger: s.logger})
	handler.Handle("/messages/identicons", &identiconHandler{logger: s.logger})
	s.server = &http.Server{Handler: handler}
	go func() {
		err := s.server.Serve(listener)
		if err != nil {
			s.logger.Error("failed to start server", zap.Error(err))
			return
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	return s.server.Shutdown(context.Background())
}

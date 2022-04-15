package server

import (
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/images"
)

type HandlerPatternMap map[string]http.HandlerFunc

func handleIdenticon(logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pks, ok := r.URL.Query()["publicKey"]
		if !ok || len(pks) == 0 {
			logger.Error("no publicKey")
			return
		}
		pk := pks[0]
		image, err := identicon.Generate(pk)
		if err != nil {
			logger.Error("could not generate identicon")
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Expires", time.Now().AddDate(60, 0, 0).Format(http.TimeFormat))

		_, err = w.Write(image)
		if err != nil {
			logger.Error("failed to write image", zap.Error(err))
		}
	}
}

func handleImage(db *sql.DB, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		messageIDs, ok := r.URL.Query()["messageId"]
		if !ok || len(messageIDs) == 0 {
			logger.Error("no messageID")
			return
		}
		messageID := messageIDs[0]
		var image []byte
		err := db.QueryRow(`SELECT image_payload FROM user_messages WHERE id = ?`, messageID).Scan(&image)
		if err != nil {
			logger.Error("failed to find image", zap.Error(err))
			return
		}
		if len(image) == 0 {
			logger.Error("empty image")
			return
		}
		mime, err := images.ImageMime(image)
		if err != nil {
			logger.Error("failed to get mime", zap.Error(err))
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(image)
		if err != nil {
			logger.Error("failed to write image", zap.Error(err))
		}
	}
}

func handleAudio(db *sql.DB, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		messageIDs, ok := r.URL.Query()["messageId"]
		if !ok || len(messageIDs) == 0 {
			logger.Error("no messageID")
			return
		}
		messageID := messageIDs[0]
		var audio []byte
		err := db.QueryRow(`SELECT audio_payload FROM user_messages WHERE id = ?`, messageID).Scan(&audio)
		if err != nil {
			logger.Error("failed to find image", zap.Error(err))
			return
		}
		if len(audio) == 0 {
			logger.Error("empty audio")
			return
		}

		w.Header().Set("Content-Type", "audio/aac")
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(audio)
		if err != nil {
			logger.Error("failed to write audio", zap.Error(err))
		}
	}
}

func handleIPFS(downloader *ipfs.Downloader, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hashes, ok := r.URL.Query()["hash"]
		if !ok || len(hashes) == 0 {
			logger.Error("no hash")
			return
		}

		_, download := r.URL.Query()["download"]

		content, err := downloader.Get(hashes[0], download)
		if err != nil {
			logger.Error("could not download hash", zap.Error(err))
			return
		}

		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Expires", time.Now().AddDate(60, 0, 0).Format(http.TimeFormat))

		_, err = w.Write(content)
		if err != nil {
			logger.Error("failed to write ipfs resource", zap.Error(err))
		}
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello I like to be a tls server. " + time.Now().String()))
}

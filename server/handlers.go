package server

import (
	"bytes"
	"database/sql"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/identity/ring"

	"go.uber.org/zap"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/images"
)

const (
	basePath       = "/messages"
	identiconsPath = basePath + "/identicons"
	imagesPath     = basePath + "/images"
	audioPath      = basePath + "/audio"
        identityImagesPath = "/identityImages"
	ipfsPath       = "/ipfs"

	// Handler routes for pairing
	pairingBase    = "/pairing"
	pairingSend    = pairingBase + "/send"
	pairingReceive = pairingBase + "/receive"

	drawRingPath        = basePath + "/drawRing"
	DrawRingTypeAccount = "account"
	DrawRingTypeContact = "contact"
)

type HandlerPatternMap map[string]http.HandlerFunc

func drawRingForAccountIdentity(multiaccountsDB *multiaccounts.Database, publicKey string, imageName string, theme ring.Theme, colorHash [][]int) ([]byte, error) {
	image, err := multiaccountsDB.GetIdentityImage(publicKey, imageName)
	if err != nil {
		return nil, err
	}
	return ring.DrawRing(&ring.DrawRingParam{
		Theme: theme, ColorHash: colorHash, ImageBytes: image.Payload, Height: image.Height, Width: image.Width,
	})
}

func drawRingForContactIdentity(db *sql.DB, publicKey string, imageName string, theme ring.Theme, colorHash [][]int) ([]byte, error) {
	var payload []byte
	err := db.QueryRow(`SELECT payload FROM chat_identity_contacts WHERE contact_id = ? and image_type = ?`, publicKey, imageName).Scan(&payload)
	if err != nil {
		return nil, err
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return ring.DrawRing(&ring.DrawRingParam{
		Theme: theme, ColorHash: colorHash, ImageBytes: payload, Height: config.Height, Width: config.Width,
	})
}

func handleDrawRing(db *sql.DB, multiaccountsDB *multiaccounts.Database, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		pks, ok := params["publicKey"]
		if !ok || len(pks) == 0 {
			logger.Error("no publicKey")
			return
		}
		types, ok := params["type"] // account or contact
		if !ok || len(types) == 0 {
			logger.Error("no type")
			return
		}
		imageNames, ok := params["imageName"]
		if !ok || len(imageNames) == 0 {
			logger.Error("no imageName")
			return
		}
		colorHash, err := colorhash.GenerateFor(pks[0])
		if err != nil {
			logger.Error("could not generate color hash")
			return
		}

		var image []byte
		var theme = getTheme(params, logger)
		if types[0] == DrawRingTypeAccount {
			image, err = drawRingForAccountIdentity(multiaccountsDB, pks[0], imageNames[0], theme, colorHash)
			if err != nil {
				logger.Error("failed to draw ring for account identity", zap.Error(err))
				return
			}
		} else if types[0] == DrawRingTypeContact {
			image, err = drawRingForContactIdentity(db, pks[0], imageNames[0], theme, colorHash)
			if err != nil {
				logger.Error("failed to draw ring for contact identity", zap.Error(err))
				return
			}
		} else {
			logger.Error("unknown type:", zap.String("type", types[0]))
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

func getTheme(params url.Values, logger *zap.Logger) ring.Theme {
	theme := ring.LightTheme // default
	themes, ok := params["theme"]
	if ok && len(themes) > 0 {
		t, err := strconv.Atoi(themes[0])
		if err != nil {
			logger.Error("invalid param[theme], value: " + themes[0])
		} else {
			theme = ring.Theme(t)
		}
	}
	return theme
}

func handleIdenticon(logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		pks, ok := params["publicKey"]
		if !ok || len(pks) == 0 {
			logger.Error("no publicKey")
			return
		}
		pk := pks[0]
		image, err := identicon.Generate(pk)
		if err != nil {
			logger.Error("could not generate identicon")
		}

		if image != nil {
			colorHash, err := colorhash.GenerateFor(pk)
			if err != nil {
				logger.Error("could not generate color hash")
				return
			}

			theme := getTheme(params, logger)
			image, err = ring.DrawRing(&ring.DrawRingParam{
				Theme: theme, ColorHash: colorHash, ImageBytes: image, Height: identicon.Height, Width: identicon.Width,
			})
			if err != nil {
				logger.Error("failed to draw ring", zap.Error(err))
			}
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

// TODO: return error
func handleIdentityImage(db *sql.DB,   logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	pks, ok := params["publicKey"]
		if !ok || len(pks) == 0 {
			logger.Error("no publicKey")
			return
		}
		pk := pks[0]
		colorHash, err := colorhash.GenerateFor(pk)
		if err != nil {
			logger.Error("could not generate color hash")
			return
		}


                rows, err := db.Query(`SELECT image_type, payload FROM chat_identity_contacts WHERE contact_id = ?`, pk)
                if err != nil {
                  logger.Error("could not fetch identity images")
                  return
                }

                defer rows.Close()

                var byteImage []byte
			theme := getTheme(params, logger)
                for rows.Next() {
                var imageType sql.NullString
                var imagePayload []byte
                  err := rows.Scan(
                    &imageType,
                    &imagePayload,
                  )
                  if err != nil {
                    logger.Error("could not scan image row")
                    return
                  }

                  config, _, err := image.DecodeConfig(bytes.NewReader(imagePayload))
                  if err != nil {
                    logger.Error("could not decode config")
                    return
                  }


                  byteImage, err = ring.DrawRing(&ring.DrawRingParam{
                    Theme: theme, ColorHash: colorHash, ImageBytes: imagePayload, Height: config.Height, Width: config.Width,
                  })
                  if err != nil {
                    logger.Error("could not generate ring")
                    return
                  }

                }

                if len(byteImage) == 0  {

		byteImage, err = identicon.Generate(pk)
                if err != nil {
			logger.Error("could not generate identicon")
                        return
                      }
			byteImage, err = ring.DrawRing(&ring.DrawRingParam{
				Theme: theme, ColorHash: colorHash, ImageBytes: byteImage, Height: identicon.Height, Width: identicon.Width,
			})
			if err != nil {
				logger.Error("failed to draw ring", zap.Error(err))
			}

                      }

                      mime, err := images.ImageMime(byteImage)
                      if err != nil {
                        logger.Error("failed to get mime", zap.Error(err))
                        return
                      }

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Expires", time.Now().AddDate(60, 0, 0).Format(http.TimeFormat))

		_, err = w.Write(byteImage)
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

func handlePairingReceive(ps *PairingServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			ps.logger.Error("ioutil.ReadAll(r.Body)", zap.Error(err))
		}

		err = ps.payload.Receive(payload)
		if err != nil {
			ps.logger.Error("ps.payload.Receive(payload)", zap.Error(err))
		}
	}
}

func handlePairingSend(ps *PairingServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err := w.Write(ps.payload.ToSend())
		if err != nil {
			ps.logger.Error("w.Write(ps.payload.ToSend())", zap.Error(err))
		}
	}
}

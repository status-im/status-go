package server

import (
	"bytes"
	"database/sql"
	"image"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/identity/ring"
)

const (
	basePath               = "/messages"
	identiconsPath         = basePath + "/identicons"
	imagesPath             = basePath + "/images"
	audioPath              = basePath + "/audio"
	ipfsPath               = "/ipfs"
	discordAuthorsPath     = "/discord/authors"
	discordAttachmentsPath = basePath + "/discord/attachments"

	// Handler routes for pairing
	accountImagesPath = "/accountImages"
	contactImagesPath = "/contactImages"
	generateQRCode    = "/GenerateQRCode"
)

type HandlerPatternMap map[string]http.HandlerFunc

func handleRequestDBMissing(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Error("can't handle media request without appdb")
	}
}

func handleRequestDownloaderMissing(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Error("can't handle media request without ipfs downloader")
	}
}

func handleAccountImages(multiaccountsDB *multiaccounts.Database, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()

		keyUids, ok := params["keyUid"]
		if !ok || len(keyUids) == 0 {
			logger.Error("no keyUid")
			return
		}
		imageNames, ok := params["imageName"]
		if !ok || len(imageNames) == 0 {
			logger.Error("no imageName")
			return
		}

		identityImage, err := multiaccountsDB.GetIdentityImage(keyUids[0], imageNames[0])
		if err != nil {
			logger.Error("handleAccountImages: failed to load image.", zap.String("keyUid", keyUids[0]), zap.String("imageName", imageNames[0]), zap.Error(err))
			return
		}

		var payload = identityImage.Payload

		if ringEnabled(params) {
			account, err := multiaccountsDB.GetAccount(keyUids[0])

			if err != nil {
				logger.Error("handleAccountImages: failed to GetAccount .", zap.String("keyUid", keyUids[0]), zap.Error(err))
				return
			}

			accColorHash := account.ColorHash

			if accColorHash == nil {
				pks, ok := params["publicKey"]
				if !ok || len(pks) == 0 {
					logger.Error("no publicKey")
					return
				}

				accColorHash, err = colorhash.GenerateFor(pks[0])
				if err != nil {
					logger.Error("could not generate color hash")
					return
				}
			}

			var theme = getTheme(params, logger)

			payload, err = ring.DrawRing(&ring.DrawRingParam{
				Theme: theme, ColorHash: accColorHash, ImageBytes: identityImage.Payload, Height: identityImage.Height, Width: identityImage.Width,
			})

			if err != nil {
				logger.Error("failed to draw ring for account identity", zap.Error(err))
				return
			}
		}

		if len(payload) == 0 {
			logger.Error("empty image")
			return
		}
		mime, err := images.GetProtobufImageMime(payload)
		if err != nil {
			logger.Error("failed to get mime", zap.Error(err))
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(payload)
		if err != nil {
			logger.Error("failed to write image", zap.Error(err))
		}
	}
}

func handleContactImages(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	if db == nil {
		return handleRequestDBMissing(logger)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		pks, ok := params["publicKey"]
		if !ok || len(pks) == 0 {
			logger.Error("no publicKey")
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

		var payload []byte
		err = db.QueryRow(`SELECT payload FROM chat_identity_contacts WHERE contact_id = ? and image_type = ?`, pks[0], imageNames[0]).Scan(&payload)
		if err != nil {
			logger.Error("failed to load image.", zap.String("contact id", pks[0]), zap.String("image type", imageNames[0]), zap.Error(err))
			return
		}

		if ringEnabled(params) {
			var theme = getTheme(params, logger)
			config, _, err := image.DecodeConfig(bytes.NewReader(payload))
			if err != nil {
				logger.Error("failed to decode config.", zap.String("contact id", pks[0]), zap.String("image type", imageNames[0]), zap.Error(err))
				return
			}

			payload, err = ring.DrawRing(&ring.DrawRingParam{
				Theme: theme, ColorHash: colorHash, ImageBytes: payload, Height: config.Height, Width: config.Width,
			})

			if err != nil {
				logger.Error("failed to draw ring for contact image.", zap.Error(err))
				return
			}
		}

		if len(payload) == 0 {
			logger.Error("empty image")
			return
		}
		mime, err := images.GetProtobufImageMime(payload)
		if err != nil {
			logger.Error("failed to get mime", zap.Error(err))
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(payload)
		if err != nil {
			logger.Error("failed to write image", zap.Error(err))
		}
	}
}

func ringEnabled(params url.Values) bool {
	addRings, ok := params["addRing"]
	return ok && len(addRings) == 1 && addRings[0] == "1"
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

func handleIdenticon(logger *zap.Logger) http.HandlerFunc {
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

		if image != nil && ringEnabled(params) {
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

func handleDiscordAuthorAvatar(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	if db == nil {
		return handleRequestDBMissing(logger)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		authorIDs, ok := r.URL.Query()["authorId"]
		if !ok || len(authorIDs) == 0 {
			logger.Error("no authorIDs")
			return
		}
		authorID := authorIDs[0]

		var image []byte
		err := db.QueryRow(`SELECT avatar_image_payload FROM discord_message_authors WHERE id = ?`, authorID).Scan(&image)
		if err != nil {
			logger.Error("failed to find image", zap.Error(err))
			return
		}
		if len(image) == 0 {
			logger.Error("empty image")
			return
		}
		mime, err := images.GetProtobufImageMime(image)
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

func handleDiscordAttachment(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	if db == nil {
		return handleRequestDBMissing(logger)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		messageIDs, ok := r.URL.Query()["messageId"]
		if !ok || len(messageIDs) == 0 {
			logger.Error("no messageID")
			return
		}
		attachmentIDs, ok := r.URL.Query()["attachmentId"]
		if !ok || len(attachmentIDs) == 0 {
			logger.Error("no attachmentID")
			return
		}
		messageID := messageIDs[0]
		attachmentID := attachmentIDs[0]
		var image []byte
		err := db.QueryRow(`SELECT payload FROM discord_message_attachments WHERE discord_message_id = ? AND id = ?`, messageID, attachmentID).Scan(&image)
		if err != nil {
			logger.Error("failed to find image", zap.Error(err))
			return
		}
		if len(image) == 0 {
			logger.Error("empty image")
			return
		}
		mime, err := images.GetProtobufImageMime(image)
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

func handleImage(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	if db == nil {
		return handleRequestDBMissing(logger)
	}

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
		mime, err := images.GetProtobufImageMime(image)
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

func handleAudio(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	if db == nil {
		return handleRequestDBMissing(logger)
	}

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

func handleIPFS(downloader *ipfs.Downloader, logger *zap.Logger) http.HandlerFunc {
	if downloader == nil {
		return handleRequestDownloaderMissing(logger)
	}

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

func handleQRCodeGeneration(multiaccountsDB *multiaccounts.Database, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()

		payload := generateQRBytes(params, logger, multiaccountsDB)
		mime, err := images.GetProtobufImageMime(payload)

		if err != nil {
			logger.Error("could not generate image from payload", zap.Error(err))
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(payload)

		if err != nil {
			logger.Error("failed to write image", zap.Error(err))
		}
	}
}

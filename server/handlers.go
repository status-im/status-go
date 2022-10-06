package server

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"go.uber.org/zap"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/identity/ring"
	"github.com/status-im/status-go/protocol/images"
	"github.com/status-im/status-go/signal"
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
	pairingBase      = "/pairing"
	pairingSend      = pairingBase + "/send"
	pairingReceive   = pairingBase + "/receive"
	pairingChallenge = pairingBase + "/challenge"

	// Session names
	sessionChallenge = "challenge"
	sessionBlocked   = "blocked"

	accountImagesPath = "/accountImages"
	contactImagesPath = "/contactImages"
)

type HandlerPatternMap map[string]http.HandlerFunc

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
			pks, ok := params["publicKey"]
			if !ok || len(pks) == 0 {
				logger.Error("no publicKey")
				return
			}
			colorHash, err := colorhash.GenerateFor(pks[0])
			if err != nil {
				logger.Error("could not generate color hash")
				return
			}

			var theme = getTheme(params, logger)

			payload, err = ring.DrawRing(&ring.DrawRingParam{
				Theme: theme, ColorHash: colorHash, ImageBytes: identityImage.Payload, Height: identityImage.Height, Width: identityImage.Width,
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
		mime, err := images.ImageMime(payload)
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
		mime, err := images.ImageMime(payload)
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

func handleDiscordAttachment(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
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
		mime, err := images.ImageMime(image)
		if err != nil {
			logger.Error("failed to get mime", zap.Error(err))
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "no-store")

		_, err = w.Write(image)
		if err != nil {
			fmt.Println("FAILED TO WRITE IMAGE: ", err)
			logger.Error("failed to write image", zap.Error(err))
			return
		}
		fmt.Println("NO ERROR WRITING IMAGE: ", r.URL)
	}
}

func handleImage(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
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

func handleAudio(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
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

func handlePairingReceive(ps *PairingServer) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess})

	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err})
			ps.logger.Error("ioutil.ReadAll(r.Body)", zap.Error(err))
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess})

		err = ps.PayloadManager.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err})
			ps.logger.Error("ps.PayloadManager.Receive(payload)", zap.Error(err))
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess})
	}
}

func handlePairingSend(ps *PairingServer) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess})

	// TODO lock sending after one successful transfer, perhaps perform the lock on the PayloadManager level
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err := w.Write(ps.PayloadManager.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err})
			ps.logger.Error("w.Write(ps.PayloadManager.ToSend())", zap.Error(err))
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess})
	}
}

func challengeMiddleware(ps *PairingServer, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := ps.cookieStore.Get(r, sessionChallenge)
		if err != nil {
			ps.logger.Error("ps.cookieStore.Get(r, pairingStoreChallenge)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
		}

		blocked, ok := s.Values[sessionBlocked].(bool)
		if ok && blocked {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// If the request header doesn't include a challenge don't punish the client, just throw a 403
		pc := r.Header.Get(sessionChallenge)
		if pc == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		c, err := common.Decrypt(base58.Decode(pc), ps.ek)
		if err != nil {
			ps.logger.Error("c, err := common.Decrypt(rc, ps.ek)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		// If the challenge is not in the session store don't punish the client, just throw a 403
		challenge, ok := s.Values[sessionChallenge].([]byte)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Only if we have both a challenge in the session store and in the request header
		// do we entertain blocking the client. Because then we know someone is trying to be sneaky.
		if !bytes.Equal(c, challenge) {
			s.Values[sessionBlocked] = true
			err = s.Save(r, w)
			if err != nil {
				ps.logger.Error("err = s.Save(r, w)", zap.Error(err))
			}

			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handlePairingChallenge(ps *PairingServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := ps.cookieStore.Get(r, sessionChallenge)
		if err != nil {
			ps.logger.Error("ps.cookieStore.Get(r, pairingStoreChallenge)", zap.Error(err))
		}

		var challenge []byte
		challenge, ok := s.Values[sessionChallenge].([]byte)
		if !ok {
			challenge = make([]byte, 64)
			_, err = rand.Read(challenge)
			if err != nil {
				ps.logger.Error("_, err = rand.Read(auth)", zap.Error(err))
			}

			s.Values[sessionChallenge] = challenge
			err = s.Save(r, w)
			if err != nil {
				ps.logger.Error("err = s.Save(r, w)", zap.Error(err))
			}
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write(challenge)
		if err != nil {
			ps.logger.Error("_, err = w.Write(challenge)", zap.Error(err))
		}
	}
}

package smartcard

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform"
	"github.com/status-im/status-go/smartcard/hexutils"
)

var (
	sdaid   = []byte{0xa0, 0x00, 0x00, 0x01, 0x51, 0x00, 0x00, 0x00}
	testKey = []byte{0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f}
)

type Installer struct {
	c Channel
}

func NewInstaller(t Transmitter) *Installer {
	return &Installer{
		c: NewNormalChannel(t),
	}
}

func (i *Installer) Install() error {
	// select / discover
	l("sending select", nil)
	sel := globalplatform.NewCommandSelect(sdaid)
	resp, err := i.send("select", sel)
	if err != nil {
		return err
	}

	// initialize update
	l("initialize update", nil)
	hostChallenge, err := generateHostChallenge()
	if err != nil {
		return err
	}

	init := globalplatform.NewCommandInitializeUpdate(hostChallenge)
	resp, err = i.send("initialize update", init)
	if err != nil {
		return err
	}

	// verify cryptogram and initialize session keys
	keys := globalplatform.NewKeyProvider(testKey, testKey)
	session, err := globalplatform.NewSession(keys, resp, hostChallenge)
	if err != nil {
		return err
	}

	i.c = NewSecureChannel(session, i.c)

	// external authenticate
	encKey := session.KeyProvider().Enc()
	extAuth, err := globalplatform.NewCommandExternalAuthenticate(encKey, session.CardChallenge(), hostChallenge)
	if err != nil {
		return err
	}

	resp, err = i.send("external authenticate", extAuth)
	if err != nil {
		return err
	}

	return nil
}

func (i *Installer) send(description string, cmd *apdu.Command, allowedResponses ...uint16) (*apdu.Response, error) {
	resp, err := i.c.Send(cmd)
	if err != nil {
		return nil, err
	}

	if len(allowedResponses) == 0 {
		allowedResponses = []uint16{apdu.SwOK}
	}

	for _, code := range allowedResponses {
		if code == resp.Sw {
			return resp, nil
		}
	}

	err = errors.New(fmt.Sprintf("unexpected response from command %s: %x", description, resp.Sw))

	return nil, err
}

func generateHostChallenge() ([]byte, error) {
	c := make([]byte, 8)
	_, err := rand.Read(c)
	return c, err
}

func l(message string, raw []byte) {
	fmt.Printf("%s %s\n", message, hexutils.BytesToHexWithSpaces(raw))
}

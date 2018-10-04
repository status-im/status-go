package lightwallet

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform"
	"github.com/status-im/status-go/smartcard/hexutils"
)

var (
	sdaid   = []byte{0xa0, 0x00, 0x00, 0x01, 0x51, 0x00, 0x00, 0x00}
	testKey = []byte{0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f}

	statusPkgAID    = []byte{0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x57, 0x61, 0x6C, 0x6C, 0x65, 0x74}
	statusAppletAID = []byte{0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x57, 0x61, 0x6C, 0x6C, 0x65, 0x74, 0x41, 0x70, 0x70}
)

type Installer struct {
	c Channel
}

func NewInstaller(t Transmitter) *Installer {
	return &Installer{
		c: NewNormalChannel(t),
	}
}

func (i *Installer) Install(capFile *os.File) (*Secrets, error) {
	sel := globalplatform.NewCommandSelect(sdaid)
	resp, err := i.send("select", sel)
	if err != nil {
		return nil, err
	}

	// initialize update
	hostChallenge, err := generateHostChallenge()
	if err != nil {
		return nil, err
	}

	init := globalplatform.NewCommandInitializeUpdate(hostChallenge)
	resp, err = i.send("initialize update", init)
	if err != nil {
		return nil, err
	}

	// verify cryptogram and initialize session keys
	keys := globalplatform.NewKeyProvider(testKey, testKey)
	session, err := globalplatform.NewSession(keys, resp, hostChallenge)
	if err != nil {
		return nil, err
	}

	i.c = NewSecureChannel(session, i.c)

	// external authenticate
	encKey := session.KeyProvider().Enc()
	extAuth, err := globalplatform.NewCommandExternalAuthenticate(encKey, session.CardChallenge(), hostChallenge)
	if err != nil {
		return nil, err
	}

	resp, err = i.send("external authenticate", extAuth)
	if err != nil {
		return nil, err
	}

	// delete current pkg and applet
	aids := [][]byte{
		statusAppletAID,
		statusPkgAID,
	}

	for _, aid := range aids {
		del := globalplatform.NewCommandDelete(aid)
		resp, err = i.send("delete", del, globalplatform.SwOK, globalplatform.SwReferencedDataNotFound)
		if err != nil {
			return nil, err
		}
	}

	// install for load
	preLoad := globalplatform.NewCommandInstallForLoad(statusPkgAID, sdaid)
	resp, err = i.send("install for load", preLoad)
	if err != nil {
		return nil, err
	}

	// load
	load, err := globalplatform.NewLoadCommandStream(capFile)
	if err != nil {
		return nil, err
	}

	for load.Next() {
		cmd := load.GetCommand()
		resp, err = i.send(fmt.Sprintf("load %d", load.Index()), cmd)
		if err != nil {
			return nil, err
		}
	}

	// install for install
	secrets, err := NewSecrets()
	if err != nil {
		return nil, err
	}

	params := []byte(secrets.Puk())
	params = append(params, secrets.PairingToken()...)

	install := globalplatform.NewCommandInstallForInstall(statusPkgAID, statusAppletAID, statusAppletAID, params)
	resp, err = i.send("install for install", install)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func (i *Installer) send(description string, cmd *apdu.Command, allowedResponses ...uint16) (*apdu.Response, error) {
	fmt.Printf("-------------------------\nsending %s\n", description)
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

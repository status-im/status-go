package protocol

import (
	"bytes"
	"fmt"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type QROptions struct {
	URL                  string `json:"url"`
	ErrorCorrectionLevel string `json:"errorCorrectionLevel"`
	Capacity             string `json:"capacity"`
	AllowProfileImage    bool   `json:"withLogo"`
}


type WriterCloserByteBuffer struct {
	*bytes.Buffer
}

func (wc WriterCloserByteBuffer) Close() error {
	return nil
}

func NewWriterCloserByteBuffer() *WriterCloserByteBuffer {
	return &WriterCloserByteBuffer{bytes.NewBuffer([]byte{})}
}

func (m *Messenger) MakeQRWithOptions(options *requests.QROptions) error {
	var imageFolderBasePath = "../_assets/tests/"
	var logoFileStaticPath = imageFolderBasePath + "logo.png"
	var QRFileNameWithLogo = imageFolderBasePath + "LogoWithQR.png"
	var QRFileNameWithOutLogo = imageFolderBasePath + "LogoWithoutQR.png"
	var writerObject qrcode.Writer

	qrc, err := qrcode.NewWith(options.URL,
		qrcode.WithEncodingMode(qrcode.EncModeAuto),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionMedium),
	)
	if err != nil {
		panic(err)
	}

	if options.AllowProfileImage {
		writerObject, err = standard.New(
			QRFileNameWithLogo,
			standard.WithLogoImageFilePNG(logoFileStaticPath),
		)
	} else {
		writerObject, err = standard.New(
			QRFileNameWithOutLogo,
		)
	}

	if err = qrc.Save(writerObject); err != nil {
		panic(err)
	}

	// we need to take the file path and make a media url over here and share that url over to status-mobile
	return err
}

// known issue : if the url of the qr is too small,
// the logo will not be superimposed on top of the QR
// example url: "https://status.im"
// so far a QR code of 27 characters works anything below that does not produce the logo overlapping on the QR.
// example url that works : "github.com/yeqown/go-qrcode"

func (m *Messenger) MakeQRCodeFromURL(URL string) ([]byte, error) {

	buf := NewWriterCloserByteBuffer()

	qrc, err := qrcode.New(URL)
	if err != nil {
		fmt.Printf("could not generate QRCode: %v", err)
		return nil, nil
	}

	w := standard.NewWithWriter(buf)
	if err != nil {
		fmt.Printf("standard.New failed: %v", err)
		return nil, nil
	}

	if err = qrc.Save(w); err != nil {
		fmt.Printf("could not save image: %v", err)
	}

	return buf.Bytes(), err

	// todo: move the QR generation logic over to the media server handler
	// on the basis of a query string we decide which image to superimpose and other options etc
	// I have no idea how we will decide which image to superimpose, perhaps a key which points to base64
	// data in the database.?

}

package protocol

import (
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type QROptions struct {
	URL                  string `json:"url"`
	ErrorCorrectionLevel string `json:"errorCorrectionLevel"`
	Capacity             string `json:"capacity"`
	AllowProfileImage    bool   `json:"withLogo"`
}

func (m *Messenger) MakeQRWithOptions(options QROptions) error {
	var logoFileStaticPath = "./logo.png"
	var imageFolderBasePath = "../_assets/tests/"
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

func (m *Messenger) MakeQRCodeFromURL(URL string, PublicKey string) error {

	//. need to figure out how to pass the image to superimpose on top of the QR code.
	//	-> The image comes from status-go So I believe we could call the status-go method here only
	//  -> To get user Image url we require the public key
	//  -> I also recently found out that the user info is stored in the database(SQLite)

	//testing QR generation with Hardcoded paths for now
	//var profileUrlPath string = "../img.png"

	// we trust that the url is actually serving a png
	// we then require the actual file because that's what this library demands

	var fileNameStaticPath string = "./img.png"

	//var profileUrlPath string = "https://cdn-icons-png.flaticon.com/512/3135/3135768.png"
	//fileName := "profile-pic.png"
	//f, err := os.Create(fileName)
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//defer f.Close()
	//
	//res, err := http.Get(profileUrlPath)
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//defer res.Body.Close()
	//
	//_, err = io.Copy(f, res.Body)
	//
	//if err != nil {
	//	panic(err)
	//}

	qrc, err := qrcode.NewWith(URL,
		qrcode.WithEncodingMode(qrcode.EncModeAuto),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionMedium),
	)
	if err != nil {
		panic(err)
	}

	//1. need to figure out how to serve the generated QR code on the media http server
	//2. once the url is ready have to return the url for the front end to consume

	w, err := standard.New(
		"./alisherOnLogo.png",
		standard.WithLogoImageFilePNG(fileNameStaticPath),
	)
	if err != nil {
		panic(err)
	}

	if err = qrc.Save(w); err != nil {
		panic(err)
	}

	return err
}

package globalplatform

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"strings"

	"github.com/status-im/status-go/smartcard/apdu"
)

var internalFiles = []string{
	"Header", "Directory", "Import", "Applet", "Class",
	"Method", "StaticField", "Export", "ConstantPool", "RefLocation",
}

// LoadCommandStream implement a struct that generates multiple Load commands used to load files to smartcards.
type LoadCommandStream struct {
	data         *bytes.Reader
	currentIndex uint8
	currentData  []byte
	p1           uint8
}

// NewLoadCommandStream returns a new LoadCommandStream to load the specified file.
func NewLoadCommandStream(file *os.File) (*LoadCommandStream, error) {
	files, err := loadFiles(file)
	if err != nil {
		return nil, err
	}

	data, err := encodeFilesData(files)
	if err != nil {
		return nil, err
	}

	return &LoadCommandStream{
		data: bytes.NewReader(data),
		p1:   P1LoadMoreBlocks,
	}, nil
}

// Next returns initialize the data for the next Load command.
// TODO:@pilu update blockSize when using encrypted data
func (lcs *LoadCommandStream) Next() bool {
	if lcs.data.Len() == 0 {
		return false
	}

	blockSize := 247 // 255 - 8 bytes for MAC
	buf := make([]byte, blockSize)
	n, err := lcs.data.Read(buf)
	if err != nil {
		return false
	}

	lcs.currentData = buf[:n]
	lcs.currentIndex++

	if lcs.data.Len() == 0 {
		lcs.p1 = P1LoadLastBlock
	}

	return true
}

// Index returns the command index.
func (lcs *LoadCommandStream) Index() uint8 {
	return lcs.currentIndex - 1
}

// GetCommand returns the current apdu command.
func (lcs *LoadCommandStream) GetCommand() *apdu.Command {
	return apdu.NewCommand(ClaGp, InsLoad, lcs.p1, lcs.Index(), lcs.currentData)
}

func loadFiles(f *os.File) (map[string][]byte, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	z, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte)

	for _, item := range z.File {
		name := strings.Split(item.FileInfo().Name(), ".")[0]
		f, err := item.Open()
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}

		files[name] = data
	}

	return files, nil
}

func encodeFilesData(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer

	for _, name := range internalFiles {
		if data, ok := files[name]; ok {
			buf.Write(data)
		}
	}

	filesData := buf.Bytes()
	length := encodeLength(len(filesData))

	data := make([]byte, 0)
	data = append(data, tagLoadFileDataBlock)
	data = append(data, length...)
	data = append(data, filesData...)

	return data, nil
}

func encodeLength(length int) []byte {
	if length < 0x80 {
		return []byte{byte(length)}
	}

	if length < 0xFF {
		return []byte{
			byte(0x81),
			byte(length),
		}
	}

	if length < 0xFFFF {
		return []byte{
			byte(0x82),
			byte((length & 0xFF00) >> 8),
			byte(length & 0xFF),
		}
	}

	return []byte{
		byte(0x83),
		byte((length & 0xFF0000) >> 16),
		byte((length & 0xFF00) >> 8),
		byte(length & 0xFF),
	}
}

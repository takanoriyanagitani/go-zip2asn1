package zip2asn1

import (
	"archive/zip"
	"bytes"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var ErrNotFound error = errors.New("not found")

type ZipItem struct {
	RawContent []byte
	zip.FileHeader
}

// Guesses the type of this item.
//
//	| size | suffix | type |
//	|:----:|:------:|:----:|
//	| 0    | /      | dir  |
//	| >0   |        | file |
//	| 0    |        | link |
func (z ZipItem) ToFileType() FileType {
	var sz int = len(z.RawContent)
	var nonZero bool = 0 < sz
	var name string = z.FileHeader.Name
	var isDir bool = strings.HasSuffix(name, "/")

	if nonZero {
		return FileTypeRegular
	}

	switch isDir {
	case true:
		return FileTypeDirectory
	default:
		return FileTypeSymlink
	}
}

func (z ZipItem) ToAsn1() Asn1ZipFile { return ItemToAsn1(z) }

type CompressionMethod asn1.Enumerated

const (
	CompressionMethodUnspecified CompressionMethod = 0
	CompressionMethodStore       CompressionMethod = 1
	CompressionMethodDeflate     CompressionMethod = 2
)

func FromZipMethod(zm uint16) CompressionMethod {
	switch zm {
	case zip.Store:
		return CompressionMethodStore
	case zip.Deflate:
		return CompressionMethodDeflate
	default:
		return CompressionMethodUnspecified
	}
}

type FileType asn1.Enumerated

const (
	FileTypeUnknown   FileType = 0
	FileTypeRegular   FileType = 1
	FileTypeDirectory FileType = 2
	FileTypeSymlink   FileType = 3
)

// Unixtime in seconds.
type Unixtime int64

type Asn1ZipFile struct {
	Name             string `asn1:"utf8"`
	Comment          string `asn1:"utf8"`
	ExtraHeader      []byte
	RawContent       []byte
	CompressedSize   int64
	UncompressedSize int64
	Modified         Unixtime
	CRC32            int32
	Method           CompressionMethod
	FileType
}

func (a Asn1ZipFile) ToDer() ([]byte, error) {
	return asn1.Marshal(a)
}

func ItemToAsn1(item ZipItem) Asn1ZipFile {
	return Asn1ZipFile{
		Name:             item.FileHeader.Name,
		Comment:          item.FileHeader.Comment,
		ExtraHeader:      item.FileHeader.Extra,
		RawContent:       item.RawContent,
		CompressedSize:   int64(item.FileHeader.CompressedSize64),
		UncompressedSize: int64(item.FileHeader.UncompressedSize64),
		Modified:         Unixtime(item.FileHeader.Modified.Unix()),
		CRC32:            int32(item.FileHeader.CRC32),
		Method:           FromZipMethod(item.FileHeader.Method),
		FileType:         item.ToFileType(),
	}
}

type ReaderToZipFile func(*zip.Reader, string) *zip.File

var GetItemOrNil ReaderToZipFile = func(
	rdr *zip.Reader,
	name string,
) *zip.File {
	for _, f := range rdr.File {
		var hdr zip.FileHeader = f.FileHeader
		var nm string = hdr.Name
		if nm == name {
			return f
		}
	}
	return nil
}

type ZipFile struct{ *zip.File }

func (f ZipFile) ToBytes() ([]byte, error) {
	rdr, e := f.File.OpenRaw()
	if nil != e {
		return nil, e
	}

	var buf bytes.Buffer
	_, e = io.Copy(&buf, rdr)
	return buf.Bytes(), e
}

func (f ZipFile) ToZipItem() (ZipItem, error) {
	raw, e := f.ToBytes()
	return ZipItem{
		RawContent: raw,
		FileHeader: f.File.FileHeader,
	}, e
}

func (f ReaderToZipFile) GetZipItem(
	zr *zip.Reader,
	itemName string,
) (ZipItem, error) {
	var file *zip.File = f(zr, itemName)
	if nil == file {
		return ZipItem{}, fmt.Errorf("%w: %s", ErrNotFound, itemName)
	}
	return ZipFile{file}.ToZipItem()
}

type ZipItemConfig struct {
	ZipName  string
	ItemName string
	ReaderToZipFile
}

func GetZipItemFs(cfg ZipItemConfig) (ZipItem, error) {
	var filename string = cfg.ZipName
	f, e := os.Open(filename)
	if nil != e {
		return ZipItem{}, e
	}
	defer f.Close()

	stat, e := f.Stat()
	if nil != e {
		return ZipItem{}, e
	}
	var sz int64 = stat.Size()

	zr, e := zip.NewReader(f, sz)
	if nil != e {
		return ZipItem{}, e
	}

	return cfg.ReaderToZipFile.GetZipItem(zr, cfg.ItemName)
}

package pdfinfo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Extract extracts metadata from a PDF file.
func Extract(file string) (Info, error) {
	f, err := os.Open(file)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return Info{}, err
	}
	return ReadMetadata(f, fi.Size())
}

// ReadMetadata reads metadata from a PDF file.
func ReadMetadata(f io.ReaderAt, size int64) (Info, error) {
	pdfInfo := newPdfInfo(f, size)

	if err := pdfInfo.getStartXref(); err != nil {
		return Info{}, err
	}

	info, err := pdfInfo.readXref()
	if err != nil {
		return Info{}, err
	}

	return Info{dict: info}, nil
}

// A Info is a single PDF info dictionary.
type Info struct {
	dict dict
}

// Key returns the value associated with the given name key in the dictionary v.
func (i Info) Key(name name) Value {
	v, ok := i.dict[name]
	if !ok {
		return Value{object: nil}
	}

	return Value{object: v}
}

// A Value is a single PDF value, such as an integer, dictionary, or array.
// The zero Value is a PDF null
type Value struct {
	object interface{}
}

// Text returns v's string value interpreted as a “text string” (defined in the PDF spec)
// and converted to UTF-8.
// If value not string Text returns the empty string.
func (v Value) Text() string {
	x, ok := v.object.(string)
	if !ok {
		return ""
	}
	if isPDFDocEncoded(x) {
		return pdfDocDecode(x)
	}
	if isUTF16(x) {
		return Utf16Decode(x[2:])
	}
	return x
}

// A pdfInfo is a single PDF file open for reading info / metadata.
type pdfInfo struct {
	f         io.ReaderAt
	size      int64
	startXref int64
}

func newPdfInfo(f io.ReaderAt, size int64) *pdfInfo {
	return &pdfInfo{f: f, size: size}
}

func (p *pdfInfo) readObject(object objptr, objectOffset int64) (object, error) {
	buf := make([]byte, 1000)
	_, err := p.f.ReadAt(buf, objectOffset)
	if err != nil {
		return dict{}, err
	}

	found := fmt.Sprintf("%d %d obj", object.id, object.gen)
	objectIndex := bytes.Index(
		buf,
		[]byte(found),
	)
	buffReader := newBuffReader(buf)
	if err := buffReader.changeOffset(objectIndex + len(found)); err != nil {
		return dict{}, err
	}

	pdfObject, err := buffReader.readObject()
	if err != nil {
		return dict{}, err
	}

	return pdfObject, nil
}

// readXref read xref based on the pdf version:
//   - PDF 1.0 - 1.4 : keyword = xref
//   - PDF 1.5+ : keyword = unreadable
func (p *pdfInfo) readXref() (dict, error) {
	buf := make([]byte, 4)
	p.f.ReadAt(buf, p.startXref)

	// PDF 1.0 - 1.4 detected
	// read from xref table
	if string(buf) == "xref" {
		xrefTable, err := readXrefTable(p.f, p.startXref, p.size)
		if err != nil {
			return dict{}, err
		}
		infoObject, err := xrefTable.getInfo()
		if err != nil {
			return dict{}, err
		}

		infoOffset, err := xrefTable.GetObjectOffset(infoObject)
		if err != nil {
			return dict{}, err
		}

		obj, err := p.readObject(infoObject, int64(infoOffset))
		if err != nil {
			return dict{}, err
		}

		dictionary, ok := obj.(dict)
		if !ok {
			return dict{}, nil
		}

		for i, obj := range dictionary {
			// if value point to another object then read object again
			if objPointer, ok := obj.(objptr); ok {
				objOffset, err := xrefTable.GetObjectOffset(objPointer)
				if err != nil {
					return dict{}, err
				}

				obj2, err := p.readObject(objPointer, int64(objOffset))
				if err != nil {
					return dict{}, err
				}
				dictionary[i] = obj2
			}
		}

		return dictionary, nil

	}

	// PDF 1.5+ detected
	// read from xref stream or metadata stream
	xrefStream, err := readXreadStream(p.f, p.startXref, p.size)
	if err != nil {
		return dict{}, err
	}

	infoObject, infoOffset, err := xrefStream.getInfo()
	if err != nil {
		return dict{}, err
	}

	obj, err := p.readObject(infoObject, int64(infoOffset))
	if err != nil {
		return dict{}, err
	}

	dictionary, ok := obj.(dict)
	if !ok {
		return dict{}, nil
	}

	for i, obj := range dictionary {
		if objPointer, ok := obj.(objptr); ok {

			objOffset := xrefStream.GetObjectOffset(objPointer)

			obj2, err := p.readObject(objPointer, int64(objOffset))
			if err != nil {
				return dict{}, err
			}

			dictionary[i] = obj2
		}
	}
	return dictionary, nil
}

func (p *pdfInfo) getStartXref() error {
	const chunkSize = 28
	offset := p.size - chunkSize
	if offset < 0 {
		offset = 0
	}
	buf := make([]byte, chunkSize)
	p.f.ReadAt(buf, offset)

	i := findLastLine(buf, "startxref")
	if i < 0 {
		return fmt.Errorf("malformed PDF: missing final startxref")
	}
	pos := p.size - chunkSize + int64(i)
	buf = make([]byte, p.size-pos)
	p.f.ReadAt(buf, pos)

	buffRead := newBuffReader(buf)
	keyword := buffRead.readKeyword()
	if keyword != "startxref" {
		return fmt.Errorf("malformed PDF: cross-reference table not found: %v", keyword)
	}

	startxRef := buffRead.readKeyword()
	i, err := strconv.Atoi(startxRef)
	if err != nil {
		return fmt.Errorf("malformed PDF: invalid startxref value: %v", startxRef)
	}
	p.startXref = int64(i)

	return nil
}

func findLastLine(buf []byte, s string) int {
	bs := []byte(s)
	max := len(buf)
	for {
		i := bytes.LastIndex(buf[:max], bs)
		if i <= 0 || i+len(bs) >= len(buf) {
			return -1
		}
		if (buf[i-1] == '\n' || buf[i-1] == '\r') && (buf[i+len(bs)] == '\n' || buf[i+len(bs)] == '\r') {
			return i
		}
		max = i
	}
}

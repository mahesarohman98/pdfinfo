package pdfinfo

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

type xrefStream struct {
	f         io.ReaderAt
	startXref int64
	fileSize  int64
	width     [3]int
}

func (x *xrefStream) GetObjectOffset(object objptr) int {
	return int(object.id) * (x.width[0] + x.width[1] + x.width[2])
}

func (x *xrefStream) getInfo() (objptr, int, error) {
	buf := make([]byte, x.fileSize-x.startXref)
	_, err := x.f.ReadAt(buf, x.startXref)
	if err != nil {
		return objptr{}, 0, err
	}

	buffReader := newBuffReader(buf)

	infoObject := objptr{}
	found := 0
	for {
		ObjName := buffReader.readName()
		if ObjName == "" {
			break
		} else if ObjName == "Info" {
			infoObject, err = buffReader.readObjectPtr()
			if err != nil {
				return objptr{}, 0, err
			}
			found++
		} else if ObjName == "W" {
			values := buffReader.readArray()
			if len(values) != 3 {
				return objptr{}, 0, fmt.Errorf("error expected width: %d", len(values))
			}
			for i, v := range values {
				w, err := strconv.Atoi(v)
				if err != nil {
					return objptr{}, 0, err
				}
				x.width[i] = w
			}
			found++
			break
		}
	}

	if found != 2 {
		return objptr{}, 0, errors.New("error info object not found")
	}

	return infoObject, x.GetObjectOffset(infoObject), nil
}

func readXreadStream(f io.ReaderAt, startXref int64, size int64) (*xrefStream, error) {
	xrefStream := &xrefStream{
		f:         f,
		startXref: startXref,
		fileSize:  size,
		width:     [3]int{},
	}

	return xrefStream, nil
}

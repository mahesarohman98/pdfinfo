package pdfinfo

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

type xrefTable struct {
	f            io.ReaderAt
	sizeFile     int
	mapXrefTable []int
	startXref    int
	startTable   int
	sizeTable    int
}

func (x *xrefTable) GetObjectOffset(object objptr) (int, error) {
	if int(object.id) > x.sizeTable {
		return 0, fmt.Errorf("error get object number exceed table size: %d", object.id)
	}
	i := x.startTable + (20 * int(object.id))
	buf := make([]byte, 20)
	_, err := x.f.ReadAt(buf, int64(x.startXref+i))
	if err != nil {
		return 0, err
	}

	byteRead := newBuffReader(buf)
	keyword := byteRead.readKeyword()
	objNumb, err := strconv.Atoi(keyword)
	if err != nil {
		return 0, err
	}

	return objNumb, nil
}

func (x *xrefTable) getInfo() (objptr, error) {
	endOfTable := x.startTable + (x.sizeTable * 20)
	bufTrailer := make([]byte, x.sizeFile-x.startXref-endOfTable)
	_, err := x.f.ReadAt(bufTrailer, int64(x.startXref+endOfTable))
	if err != nil {
		return objptr{}, err
	}

	buffReader := newBuffReader(bufTrailer)
	for {
		name := buffReader.readName()
		if name == "" {
			break
		}
		if name == "Info" {
			obj, err := buffReader.readObjectPtr()
			if err != nil {
				return objptr{}, err
			}
			return obj, nil
		}
	}

	return objptr{}, errors.New("error get info not found")
}

func readXrefTable(f io.ReaderAt, startXref int64, size int64) (*xrefTable, error) {
	buf := make([]byte, 20)
	_, err := f.ReadAt(buf, startXref)
	if err != nil {
		return &xrefTable{}, err
	}

	buffReader := newBuffReader(buf)
	xrefTableObj := &xrefTable{
		f:            f,
		sizeFile:     int(size),
		startXref:    int(startXref),
		mapXrefTable: []int{},
		startTable:   0,
	}

	keyword := buffReader.readKeyword()
	if keyword != "xref" {
		return &xrefTable{}, fmt.Errorf("xref table not started with xref, instead : %s", keyword)
	}

	// detect if table start
	keyword = buffReader.readKeyword()
	if keyword != "0" {
		return &xrefTable{}, fmt.Errorf("xreftable not start: %s", keyword)
	}
	// detect table size
	keyword = buffReader.readKeyword()
	tableSize, err := strconv.ParseInt(keyword, 10, 64)
	if err != nil {
		return &xrefTable{}, fmt.Errorf("error converting table size: %s", err)
	}

	xrefTableObj.startTable = buffReader.i
	xrefTableObj.sizeTable = int(tableSize)

	return xrefTableObj, nil
}

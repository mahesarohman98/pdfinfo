package pdfinfo

import (
	"fmt"
	"strconv"
)

func isSpace(c byte) bool {
	switch c {
	case '\x00', '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return false
}

func isDelim(c byte) bool {
	switch c {
	case '<', '>', '(', ')', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

type BuffReader struct {
	buf  []byte
	size int
	i    int
}

func newBuffReader(c []byte) *BuffReader {
	return &BuffReader{buf: c, size: len(c), i: 0}
}

func (br *BuffReader) changeOffset(i int) error {
	if i > len(br.buf) {
		return fmt.Errorf("error: length buffer exceed: %d", br.i)
	}

	br.i = i
	return nil
}

func (br *BuffReader) readByte() byte {
	if br.i >= br.size {
		return '\n'
	}
	c := br.buf[br.i]
	br.i++
	return c
}

func (br *BuffReader) unreadByte() {
	br.i--
}

func (br *BuffReader) readKeyword() string {
	// skip space and delim
	for {
		c := br.readByte()
		if !isSpace(c) && !isDelim(c) {
			br.unreadByte()
			break
		}
	}

	// read keyword
	tmp := []byte{}
	for {
		c := br.readByte()
		if isSpace(c) || isDelim(c) {
			break
		}
		tmp = append(tmp, c)
	}

	return string(tmp)
}

type object interface{}

type objptr struct {
	id  uint32
	gen uint16
}

type dict map[name]object

type name string

type array []string

func (br *BuffReader) readName() name {
	for {
		c := br.readByte()
		if br.i >= br.size {
			return name("")
		}
		if c == '/' {
			break
		}
	}

	tmp := []byte{}
	for {
		c := br.readByte()
		if isDelim(c) {
			break
		}
		if isSpace(c) {
			if c != ' ' {
				break
			}
			c2 := br.readByte()
			if isDelim(c2) {
				br.unreadByte()
				break
			}
			if _, err := strconv.Atoi(string(c2)); err == nil {
				br.unreadByte()
				break
			}
			tmp = append(tmp, c, c2)
			continue
		}
		tmp = append(tmp, c)
	}

	br.unreadByte()
	return name(string(tmp))
}

func (br *BuffReader) readArray() array {
	for {
		c := br.readByte()
		if br.i >= br.size {
			return array{}
		}
		if c == '[' {
			break
		}
	}

	values := array{}
	for {
		c := br.readByte()
		br.unreadByte()
		if c == ']' || isDelim(c) {
			break
		}
		values = append(values, br.readKeyword())
	}

	return values
}

func (br *BuffReader) readLiteralString() string {
	tmp := []byte{}
	depth := 1
Loop:
	for {
		if br.i > br.size-1 {
			break Loop
		}
		c := br.readByte()
		switch c {
		default:
			tmp = append(tmp, c)
		case '(':
			depth++
			tmp = append(tmp, c)
		case ')':
			if depth--; depth == 0 {
				break Loop
			}
			tmp = append(tmp, c)
		case '\\':
			switch c = br.readByte(); c {
			default:
				tmp = append(tmp, '\\', c)
			case 'n':
				tmp = append(tmp, '\n')
			case 'r':
				tmp = append(tmp, '\r')
			case 'b':
				tmp = append(tmp, '\b')
			case 't':
				tmp = append(tmp, '\t')
			case 'f':
				tmp = append(tmp, '\f')
			case '(', ')', '\\':
				tmp = append(tmp, c)
			case '\r':
				if br.buf[br.i] != '\n' {
					br.i = 0
				}
				fallthrough
			case '\n':
				// no append
			case '0', '1', '2', '3', '4', '5', '6', '7':
				x := int(c - '0')
				for i := 0; i < 2; i++ {
					c = br.readByte()
					if c < '0' || c > '7' {
						break
					}
					x = x*8 + int(c-'0')
				}

				tmp = append(tmp, byte(x))
			}
		}
	}
	return string(tmp)
}

func (br *BuffReader) readObjectPtr() (objptr, error) {
	word := br.readKeyword()
	t1, err := strconv.Atoi(word)
	if err != nil {
		return objptr{}, err
	}

	word = br.readKeyword()
	t2, err := strconv.Atoi(word)
	if err != nil {
		return objptr{}, err
	}

	word = br.readKeyword()
	if word != "R" {
		return objptr{}, err
	}

	return objptr{id: uint32(t1), gen: uint16(t2)}, nil
}

func (br *BuffReader) readObject() (object, error) {
	for {
		c := br.readByte()
		if !isSpace(c) {
			br.unreadByte()
			break
		}
	}
	tok := br.readByte()
	switch tok {
	case '[':
		br.unreadByte()
		return br.readArray(), nil
	case '<':
		if br.readByte() == '<' {
			br.changeOffset(br.i - 2)
			return br.readDict(), nil
		}
		// utf-16-be format
		// skip FEFF
		// example:
		// <FEFF00480041004C>
		br.changeOffset(br.i - 1)
		return br.readHexString(), nil
	case '(':
		return br.readLiteralString(), nil
	default:
		br.unreadByte()
		if _, err := strconv.Atoi(string(tok)); err == nil {
			return br.readObjectPtr()
		}
		return br.readKeyword(), nil

	}
}

func (br *BuffReader) readHexString() string {
	tmp := []byte{}
	for {
	Loop:
		c := br.readByte()
		if c == '>' {
			break
		}
		if isSpace(c) {
			goto Loop
		}
	Loop2:
		c2 := br.readByte()
		if isSpace(c2) {
			goto Loop2
		}
		x := unhex(c)<<4 | unhex(c2)
		if x < 0 {
			break
		}
		tmp = append(tmp, byte(x))
	}
	return string(tmp)
}

func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c) - '0'
	case 'a' <= c && c <= 'f':
		return int(c) - 'a' + 10
	case 'A' <= c && c <= 'F':
		return int(c) - 'A' + 10
	}
	return -1
}
func (br *BuffReader) readDict() dict {
	// skips until <<
	for {
		c := br.readByte()
		if br.i >= br.size {
			return dict{}
		}
		if c == '<' {
			if br.readByte() == '<' {
				break
			}
		}
	}

	dictionary := make(dict)
	for {
		if br.i >= br.size {
			return dict{}
		}

		c := br.readByte()
		if isSpace(c) {
			continue
		}
		if c == '>' {
			break
		}

		br.unreadByte()
		name := br.readName()

		obj, err := br.readObject()
		if err != nil {
			continue
		}
		dictionary[name] = obj

	}

	return dictionary
}

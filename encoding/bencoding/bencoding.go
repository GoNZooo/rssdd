package bencoding

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

type BencodeDecodingError struct {
	msg  string
	data []byte
}

func (e BencodeDecodingError) Error() string {
	return fmt.Sprintf("%s (%s)", e.msg, e.data)
}

type Bencode interface {
	Encode() ([]byte, error)
}

func DecodeValue(r *bufio.Reader) (Bencode, error) {
	leadingByte := make([]byte, 1)
	_, err := r.Read(leadingByte)
	if err != nil {
		return nil, err
	}

	switch leadingByte[0] {
	case 'i':
		i, err := decodeInt64(r)
		if err != nil {
			return nil, err
		}
		return i, nil
	case 'l':
		l, err := decodeList(r)
		if err != nil {
			return nil, err
		}
		return l, nil
	case 'd':
		d, err := decodeDict(r)
		if err != nil {
			return nil, err
		}
		return d, nil
	default:
		err = r.UnreadByte()
		if err != nil {
			return nil, err
		}
		s, err := decodeString(r)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
}

type BencodeInt64 int64

func (i BencodeInt64) Encode() ([]byte, error) {
	return []byte("i" + strconv.FormatInt(int64(i), 10) + "e"), nil
}

func decodeInt64(r *bufio.Reader) (BencodeInt64, error) {
	leadingByte := make([]byte, 1)
	_, err := r.Read(leadingByte)

	data := make([]byte, 0)
	if leadingByte[0] != 'i' {
		data = append(data, leadingByte[0])
	}
	for {
		b := make([]byte, 1)
		bytesRead, err := r.Read(b)
		if bytesRead == 0 {
			return 0, BencodeDecodingError{msg: "unexpected EOF when decoding integer", data: data}
		}
		if err != nil {
			return 0, err
		}
		if b[0] == 'e' {
			break
		}
		data = append(data, b[0])
	}

	parsed, err := strconv.ParseInt(string(data[:]), 10, 64)
	if err != nil {
		return 0, err
	}

	return BencodeInt64(parsed), nil
}

type BencodeString string

func (s BencodeString) Encode() ([]byte, error) {
	return []byte(strconv.Itoa(len(s)) + ":" + string(s)), nil
}

func decodeString(r *bufio.Reader) (BencodeString, error) {
	data := make([]byte, 0)
	for {
		b := make([]byte, 1)
		bytesRead, err := r.Read(b)
		if bytesRead == 0 {
			return "", BencodeDecodingError{msg: "unexpected EOF when decoding string", data: data}
		}
		if err != nil {
			return "", err
		}
		if b[0] == ':' {
			break
		}
		data = append(data, b[0])
	}

	length, err := strconv.Atoi(string(data[:]))
	if err != nil {
		return "", err
	}

	stringData := make([]byte, 0)
	bytesRead := 0
	for bytesRead < length {
		buffer := make([]byte, length-bytesRead)
		bytesReadNow, err := r.Read(buffer)
		if bytesReadNow == 0 {
			return "", BencodeDecodingError{msg: "unexpected EOF when decoding string", data: nil}
		}
		if err != nil {
			return "", err
		}
		bytesRead += bytesReadNow
		stringData = append(stringData, buffer[:bytesReadNow]...)
	}

	return BencodeString(string(stringData[:])), nil
}

type BencodeList []Bencode

func (l BencodeList) Encode() ([]byte, error) {
	data := make([]byte, 0)
	data = append(data, 'l')
	for _, item := range l {
		itemData, err := item.Encode()
		if err != nil {
			return nil, err
		}
		data = append(data, itemData...)
	}
	data = append(data, 'e')
	return data, nil
}

func decodeList(r *bufio.Reader) (BencodeList, error) {
	l := make(BencodeList, 0, 0)
	done := false
	for !done {
		b, err := r.ReadByte()
		if err == io.EOF {
			return nil, BencodeDecodingError{msg: "unexpected EOF when decoding list", data: nil}
		} else if err != nil {
			return nil, err
		}
		if b == 'e' {
			done = true
			break
		}
		r.UnreadByte()
		item, err := DecodeValue(r)
		if err != nil {
			return nil, err
		}
		l = append(l, item)
	}
	return l, nil
}

type BencodeDict map[string]Bencode

func (d BencodeDict) Encode() ([]byte, error) {
	data := make([]byte, 0)
	data = append(data, 'd')
	for key, value := range d {
		keyData, err := BencodeString(key).Encode()
		if err != nil {
			return nil, err
		}
		data = append(data, keyData...)
		valueData, err := value.Encode()
		if err != nil {
			return nil, err
		}
		data = append(data, valueData...)
	}
	data = append(data, 'e')
	return data, nil
}

func decodeDict(r *bufio.Reader) (BencodeDict, error) {
	d := make(BencodeDict)
	done := false
	for !done {
		b, err := r.ReadByte()
		if err == io.EOF {
			return nil, BencodeDecodingError{msg: "unexpected EOF when decoding dict", data: nil}
		} else if err != nil {
			return nil, err
		}
		if b == 'e' {
			done = true
			break
		}
		r.UnreadByte()
		key, err := decodeString(r)
		if err != nil {
			return nil, err
		}
		value, err := DecodeValue(r)
		if err != nil {
			return nil, err
		}
		d[string(key)] = value
	}
	return d, nil
}

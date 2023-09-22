package bencoding

import (
	"bufio"
	"bytes"
	"testing"
)

func TestDecodeInt64(t *testing.T) {
	data := bufio.NewReader(bytes.NewReader([]byte("i123e")))

	i, err := DecodeInt64(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if int64(i) != 123 {
		t.Errorf("expected 123, got %v", i)
	}

	data = bufio.NewReader(bytes.NewReader([]byte("i-123e")))
	i, err = DecodeInt64(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if int64(i) != -123 {
		t.Errorf("expected -123, got %v", i)
	}

	// data = bytes.NewReader([]byte("i123"))
	// i, err = DecodeInt64(data)
	// if err == nil {
	// 	t.Errorf("expected error, got %v", i)
	// }
	//
	// data = bytes.NewReader([]byte("123e"))
	// i, err = DecodeInt64(data)
	// if err != nil {
	// 	t.Errorf("error: %v", err)
	// }
	// if int64(i) != 123 {
	// 	t.Errorf("expected 123, got %v", i)
	// }
}

func TestEncodeInt64(t *testing.T) {
	i := BencodeInt64(123)
	data, err := i.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "i123e" {
		t.Errorf("expected i123e, got %v", string(data))
	}

	i = BencodeInt64(-123)
	data, err = i.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "i-123e" {
		t.Errorf("expected i-123e, got %v", string(data))
	}

	i = BencodeInt64(0)
	data, err = i.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "i0e" {
		t.Errorf("expected i0e, got %v", string(data))
	}
}

func TestDecodeString(t *testing.T) {
	data := bufio.NewReader(bytes.NewReader([]byte("4:spam")))
	s, err := DecodeString(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(s) != "spam" {
		t.Errorf("expected spam, got %v", s)
	}

	data = bufio.NewReader(bytes.NewReader([]byte("4spam")))
	s, err = DecodeString(data)
	if err == nil {
		t.Errorf("expected error, got %v", s)
	}

	data = bufio.NewReader(bytes.NewReader([]byte("6:spam")))
	s, err = DecodeString(data)
	if err == nil {
		t.Errorf("expected error, got %v", s)
	}
}

func TestEncodeString(t *testing.T) {
	s := BencodeString("spam")
	data, err := s.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "4:spam" {
		t.Errorf("expected 4:spam, got %v", string(data))
	}

	s = BencodeString("")
	data, err = s.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "0:" {
		t.Errorf("expected 0:, got %v", string(data))
	}

	s = BencodeString("spameggs")
	data, err = s.Encode()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if string(data) != "8:spameggs" {
		t.Errorf("expected 8:spameggs, got %v", string(data))
	}
}

func TestDecodeList(t *testing.T) {
	data := bufio.NewReader(bytes.NewReader([]byte("4:spam3:hame")))
	l, err := DecodeList(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if len(l) != 2 {
		t.Errorf("expected 2, got %v", len(l))
	}

	s1, ok := l[0].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[0])
	}
	if string(s1) != "spam" {
		t.Errorf("expected 'spam', got %v", l[0])
	}
	s2, ok := l[1].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[1])
	}
	if string(s2) != "ham" {
		t.Errorf("expected 'ham', got %v", l[1])
	}

	data = bufio.NewReader(bytes.NewReader([]byte("4:spam4:eggs")))
	l, err = DecodeList(data)
	if err == nil {
		t.Errorf("expected error, got %v", l)
	}
}

func TestDecodeValue(t *testing.T) {
	data := bufio.NewReader(bytes.NewReader([]byte("i123e")))
	value, err := DecodeValue(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	i, ok := value.(BencodeInt64)
	if !ok {
		t.Errorf("expected int, got %v", value)
	}
	if int64(i) != 123 {
		t.Errorf("expected 123, got %v", i)
	}

	data = bufio.NewReader(bytes.NewReader([]byte("4:spam")))
	value, err = DecodeValue(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	s, ok := value.(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", value)
	}
	if string(s) != "spam" {
		t.Errorf("expected spam, got %v", s)
	}

	data = bufio.NewReader(bytes.NewReader([]byte("l4:spam3:hame")))
	value, err = DecodeValue(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	l, ok := value.(BencodeList)
	if !ok {
		t.Errorf("expected list, got %v", value)
	}
	if len(l) != 2 {
		t.Errorf("expected 2, got %v", len(l))
	}
	s1, ok := l[0].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[0])
	}
	if string(s1) != "spam" {
		t.Errorf("expected 'spam', got %v", l[0])
	}
	s2, ok := l[1].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[1])
	}
	if string(s2) != "ham" {
		t.Errorf("expected 'ham', got %v", l[1])
	}

	data = bufio.NewReader(bytes.NewReader([]byte("d4:spaml4:spam3:hamee")))
	value, err = DecodeValue(data)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	d, ok := value.(BencodeDict)
	if !ok {
		t.Errorf("expected dict, got %v", value)
	}
	if len(d) != 1 {
		t.Errorf("expected 1, got %v", len(d))
	}
	l, ok = d["spam"].(BencodeList)
	if !ok {
		t.Errorf("expected list, got %v", d["spam"])
	}
	if len(l) != 2 {
		t.Errorf("expected 2, got %v", len(l))
	}
	s1, ok = l[0].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[0])
	}
	if string(s1) != "spam" {
		t.Errorf("expected 'spam', got %v", l[0])
	}
	s2, ok = l[1].(BencodeString)
	if !ok {
		t.Errorf("expected string, got %v", l[1])
	}
	if string(s2) != "ham" {
		t.Errorf("expected 'ham', got %v", l[1])
	}
}

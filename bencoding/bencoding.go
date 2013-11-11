// Package bencoding implements types representing, and functions for
// encoding/decoding, data in BitTorrent's Bencoding format as specified in
// BEP 3 (http://www.bittorrent.org/beps/bep_0003.html#bencoding).
package bencoding

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

type Bencodable interface {
	WriteBencodedTo(io.Writer) error
	ToJsonable() (jval interface{}, err error)
}

type Int int64
type String string
type List []Bencodable
type Dict map[String]Bencodable

func Encode(bval Bencodable) ([]byte, error) {
	buffer := new(bytes.Buffer)
	err := bval.WriteBencodedTo(buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func Decode(data []byte) (Bencodable, error) {
	var buffer bytes.Buffer

	_, err := buffer.Write(data)
	if err != nil {
		return nil, err
	}

	value, err := decodeNextFrom(&buffer)
	if err != nil {
		return nil, err
	}

	nextByte, err := buffer.ReadByte()
	if err != io.EOF {
		return nil, errors.New(fmt.Sprintf(
			"Unexpected data after end of bencoded value, starting with %v.",
			strconv.Quote(string(nextByte))))
	}

	return value, nil
}

func decodeNextFrom(buffer *bytes.Buffer) (Bencodable, error) {
	nextByte, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	buffer.UnreadByte()

	var result Bencodable

	switch nextByte {
	case 'i':
		result, err = decodeNextIntFrom(buffer)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		result, err = decodeNextStringFrom(buffer)
	case 'l':
		result, err = decodeNextListFrom(buffer)
	case 'd':
		result, err = decodeNextDictFrom(buffer)
	default:
		err = errors.New(fmt.Sprintf(
			"Unexpected initial byte in bencoded data: %v",
			strconv.Quote(string(nextByte))))
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bint Int) WriteBencodedTo(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "i%ve", int64(bint))
	return err
}

func decodeNextIntFrom(buffer *bytes.Buffer) (Int, error) {
	initial, err := buffer.ReadByte()
	if initial != 'i' || err != nil {
		panic("How is this not an integer?")
	}

	firstByte, err := buffer.ReadByte()
	if err != nil {
		return -1, err
	}

	isNegative := false

InterpretingInitial:
	for {
		switch firstByte {
		case '-':
			if !isNegative {
				isNegative = true
				firstByte, err = buffer.ReadByte()
				if err != nil {
					return -1, err
				}
				continue InterpretingInitial
			} else {
				return -1, errors.New("Unexpected \"--\" in integer value.")
			}
		case '0':
			// Leading zero is only allowed for value "i0e".
			if isNegative {
				return -1, errors.New("Unexpected \"-0\" in integer value.")
			}
			remainingByte, err := buffer.ReadByte()
			if err != nil {
				return -1, err
			}
			if remainingByte != byte('e') {
				return -1, errors.New("Unexpected leading zero in integer value.")
			}
			return 0, nil
		default:
			buffer.UnreadByte()
			break InterpretingInitial
		}
	}

	digits := []byte{}

AccumulatingDigits:
	for {
		nextByte, err := buffer.ReadByte()
		if err != nil {
			return -1, err
		}
		switch nextByte {
		case 'e':
			break AccumulatingDigits
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			digits = append(digits, nextByte)
		default:
			return -1, errors.New(fmt.Sprintf(
				"Unexpected byte in integer value: %v", nextByte))
		}
	}

	digitValue, err := strconv.ParseInt(string(digits), 10, 64)

	if err != nil {
		return -1, err
	}

	if digitValue <= 0 {
		panic("digitValue should not be able to be <= 0 here.")
	}

	if !isNegative {
		return Int(digitValue), nil
	} else {
		return Int(-digitValue), nil
	}

}

func (bstr String) WriteBencodedTo(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "%v:%v", len(bstr), string(bstr))
	return err
}

func decodeNextStringFrom(buffer *bytes.Buffer) (String, error) {
	firstByte, err := buffer.ReadByte()
	if err != nil {
		return "", err
	}

	if firstByte == '0' {
		// must be null string
		lastByte, err := buffer.ReadByte()
		switch {
		case err != nil:
			return "", err
		case lastByte != ':':
			return "", errors.New("Unexpected leading zero in non-{empty string}.")
		default:
			return "", nil
		}
	}

	buffer.UnreadByte()

	digits := []byte{}

AccumulatingDigits:
	for {
		nextByte, err := buffer.ReadByte()
		if err != nil {
			return "", err
		}
		switch nextByte {
		case ':':
			break AccumulatingDigits
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			digits = append(digits, nextByte)
		default:
			return "", errors.New(fmt.Sprintf(
				"Unexpected byte in integer value: %v",
				strconv.Quote(string(nextByte))))
		}
	}

	strLen, err := strconv.ParseInt(string(digits), 10, 64)

	if err != nil {
		return "", err
	}

	if strLen <= 0 {
		panic("strLen should not be able to be <= 0 here.")
	}

	contents := make([]byte, strLen)

	n, err := buffer.Read(contents)

	if err != nil {
		return "", err
	}
	if int64(n) != strLen {
		return "", errors.New("Declared string length greater than remaining input.")
	}

	return String(contents), nil
}

func (blist List) WriteBencodedTo(writer io.Writer) error {
	_, err := writer.Write([]byte("l"))
	if err != nil {
		return err
	}

	for _, item := range blist {
		err = item.WriteBencodedTo(writer)
		if err != nil {
			return err
		}
	}

	_, err = writer.Write([]byte("e"))
	if err != nil {
		return err
	}

	return nil
}

func decodeNextListFrom(buffer *bytes.Buffer) (List, error) {
	initial, err := buffer.ReadByte()
	if initial != 'l' || err != nil {
		panic("How is this not a list?")
	}

	result := make(List, 0)

AccumulateItems:
	for {
		nextByte, err := buffer.ReadByte()
		switch {
		case err != nil:
			return nil, err
		case nextByte == 'e':
			break AccumulateItems
		default:
			buffer.UnreadByte()
			value, err := decodeNextFrom(buffer)
			if err != nil {
				return nil, err
			}

			result = append(result, value)
		}
	}

	return result, nil
}

func (bdict Dict) WriteBencodedTo(writer io.Writer) error {
	_, err := writer.Write([]byte("d"))
	if err != nil {
		return err
	}

	strKeys := make([]string, len(bdict))

	i := 0
	for strKey, _ := range bdict {
		strKeys[i] = string(strKey)
		i += 1
	}

	sort.Strings(strKeys)

	for _, strKey := range strKeys {
		key := String(strKey)
		value := bdict[key]

		err = key.WriteBencodedTo(writer)
		if err != nil {
			return err
		}
		err = value.WriteBencodedTo(writer)
		if err != nil {
			return err
		}
	}

	_, err = writer.Write([]byte("e"))
	if err != nil {
		return err
	}

	return nil
}

func decodeNextDictFrom(buffer *bytes.Buffer) (Dict, error) {
	initial, err := buffer.ReadByte()
	if initial != 'd' || err != nil {
		panic("How is this not a dictionary?")
	}

	result := make(Dict, 0)

AccumulateItems:
	for {
		nextByte, err := buffer.ReadByte()
		switch {
		case err != nil:
			return nil, err
		case nextByte == 'e':
			break AccumulateItems
		default:
			buffer.UnreadByte()

			key, err := decodeNextStringFrom(buffer)
			if err != nil {
				return nil, err
			}

			value, err := decodeNextFrom(buffer)
			if err != nil {
				return nil, err
			}

			result[key] = value
		}
	}

	return result, nil
}

func (str String) String() string {
	return strconv.Quote(string(str))
}

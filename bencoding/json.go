package bencoding

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

// Converts Bencoded values to/from equivalent more-generic structures which
// can be encoded/decoded by encodings/json.
//
// If a JSON Number contains a non-integer value, an error is returned.
//
// There's also possible data loss when converting a bencoded 64-bit integer
// into a JSONable 64-bit float, but I haven't given thought to how that's
// handled.

func (dict Dict) ToJsonable() (jval interface{}, err error) {
	result := make(map[string]interface{})

	for key, value := range dict {
		jsonableKey, err := key.ToJsonable()
		if err != nil {
			return nil, err
		}

		jsonableKeyStr, ok := jsonableKey.(string)
		if !ok {
			panic("How isn't this a string?")
		}

		jsonableValue, err := value.ToJsonable()
		if err != nil {
			return nil, err
		}

		result[jsonableKeyStr] = jsonableValue
	}

	return result, nil
}

func (list List) ToJsonable() (jval interface{}, err error) {
	result := make([]interface{}, len(list))

	for i, value := range list {
		jsonableValue, err := value.ToJsonable()
		if err != nil {
			return nil, err
		}

		result[i] = jsonableValue
	}

	return result, nil
}

func (n Int) ToJsonable() (jval interface{}, err error) {
	return int64(n), nil
}

func (str String) ToJsonable() (jval interface{}, err error) {
	// Bencoded bytes strings are transformed into characters strings who
	// codepoints correspond to the byte value.

	encodedStringPieces := make([][]byte, 0)

	for _, byteValue := range []byte(str) {
		byteAsUtf8Char := make([]byte, 4)

		n := utf8.EncodeRune(byteAsUtf8Char, rune(byteValue))

		encodedStringPieces = append(encodedStringPieces, byteAsUtf8Char[:n])
	}

	return string(bytes.Join(encodedStringPieces, []byte(""))), nil
}

func FromJsonable(jval interface{}) (Bencodable, error) {
	switch jval := jval.(type) {
	case float64:
		if jval != float64(Int(jval)) {
			return nil, errors.New("Cannot bencode non-integer Number")
		}

		return Int(jval), nil

	case map[string]interface{}:
		bval := Dict{}

		for key, value := range jval {
			bKey, err := FromJsonable(key)
			if err != nil {
				return nil, err
			}

			bVal, err := FromJsonable(value)
			if err != nil {
				return nil, err
			}

			bval[bKey.(String)] = bVal
		}

		return bval, nil

	case []interface{}:
		bval := make(List, len(jval))

		for i, value := range jval {
			bVal, err := FromJsonable(value)
			if err != nil {
				return nil, err
			}

			bval[i] = bVal
		}

		return bval, nil

	case string:
		pieces := make([]byte, 0)

		for _, runeVal := range jval {
			if runeVal > 0xFF {
				return nil, errors.New("Error: character with codepoint > 0xFF")
			}

			pieces = append(pieces, byte(runeVal))
		}

		return String(pieces), nil

	default:
		return nil, errors.New(fmt.Sprintf("Cannot bencode %v", jval))
	}
}

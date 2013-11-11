package bencoding

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

func Test(t *testing.T) {
	testCases := map[string]Bencodable{
		"i-1e":   Int(-1),
		"i0e":    Int(0),
		"i1e":    Int(1),
		"i1023e": Int(1023),
		"li1ei2ed2:aai1e2:abl1:ci4e1:deeledee": List{
			Int(1), Int(2), Dict{
				"ab": List{String("c"), Int(4), String("d")},
				"aa": Int(1),
			}, List{}, Dict{},
		},
		"d1:ad2:id21:abcdefghij0123456789\xFFe1:q4:ping1:t2:aa1:y1:qe": Dict{
			"y": String("q"),
			"a": Dict{
				"id": String("abcdefghij0123456789\xFF"),
			},
			"t": String("aa"),
			"q": String("ping"),
		},
	}

	for originalEncodedStr, originalDecoded := range testCases {
		originalEncoded := []byte(originalEncodedStr)

		encoded, err := Encode(originalDecoded)
		if err != nil {
			t.Error("Error encoding", originalDecoded, err)
		} else {
			if !reflect.DeepEqual(encoded, originalEncoded) {
				t.Error(
					"Encoded value", strconv.Quote(string(encoded)),
					"does not equal expected value", strconv.Quote(string(originalEncoded)))
			}
		}

		decoded, err := Decode(originalEncoded)
		if err != nil {
			t.Error("Error decoding", strconv.Quote(string(originalEncoded)), err)
		} else {
			if !reflect.DeepEqual(decoded, originalDecoded) {
				t.Error(
					"Decoded value", decoded,
					"does not equal expected value", originalDecoded)
			}
		}

		jsonable, err := decoded.ToJsonable()
		if err != nil {
			t.Error("Error JSONablizing")
		}

		jsoned, err := json.Marshal(jsonable)
		if err != nil {
			t.Error("Error JSONing")
		}

		var unjsoned interface{}
		err = json.Unmarshal(jsoned, &unjsoned)
		if err != nil {
			t.Error("Error unJSONing")
		}

		unjsonabled, err := FromJsonable(unjsoned)
		if err != nil {
			t.Error("Error unjsonabling")
		}

		if !reflect.DeepEqual(unjsonabled, decoded) {
			t.Error(
				"JSON round-tripped value", unjsonabled,
				"does not equal expected value", decoded)
		}

	}
}

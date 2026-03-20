package service

import (
	"testing"
	"welink/backend/model"
)

func TestParseExplicitGenderFromExtraBuffer(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want model.Gender
	}{
		{name: "male", raw: []byte{0x28, 0x01}, want: model.GenderMale},
		{name: "female", raw: []byte{0x28, 0x02}, want: model.GenderFemale},
		{name: "invalid varint value", raw: []byte{0x28, 0x03}, want: model.GenderUnknown},
		{name: "conflicting markers", raw: []byte{0x28, 0x01, 0x28, 0x02}, want: model.GenderUnknown},
		{name: "malformed payload", raw: []byte{0x28}, want: model.GenderUnknown},
		{name: "empty", raw: nil, want: model.GenderUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseExplicitGenderFromExtraBuffer(tc.raw)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

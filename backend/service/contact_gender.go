package service

import (
	"encoding/binary"
	"welink/backend/model"
)

// parseExplicitGenderFromExtraBuffer parses explicit sex markers from contact.extra_buffer.
// Current data snapshots use two stable encodings:
// - legacy/tlv-like rows: field #2 varint => 1 male, 2 female
// - protobuf-like rows:  field #5 varint => 1 male, 2 female
// Any ambiguity falls back to unknown.
func parseExplicitGenderFromExtraBuffer(extra []byte) model.Gender {
	if len(extra) == 0 {
		return model.GenderUnknown
	}

	male, female, ok := parseGenderFromTopLevelProto(extra)
	if !ok {
		return model.GenderUnknown
	}
	if male == female {
		return model.GenderUnknown
	}
	if male {
		return model.GenderMale
	}
	return model.GenderFemale
}

func parseGenderFromTopLevelProto(data []byte) (male bool, female bool, ok bool) {
	ok = true
	for i := 0; i < len(data); {
		tag, n := binary.Uvarint(data[i:])
		if n <= 0 {
			return male, female, false
		}
		i += n
		field := int(tag >> 3)
		wire := int(tag & 0x7)
		if field == 0 {
			return male, female, false
		}

		switch wire {
		case 0: // varint
			val, vn := binary.Uvarint(data[i:])
			if vn <= 0 {
				return male, female, false
			}
			i += vn
			if field == 2 || field == 5 {
				if val == 1 {
					male = true
				} else if val == 2 {
					female = true
				}
			}
		case 1: // fixed64
			if i+8 > len(data) {
				return male, female, false
			}
			i += 8
		case 2: // length-delimited
			l, ln := binary.Uvarint(data[i:])
			if ln <= 0 {
				return male, female, false
			}
			i += ln
			if l > uint64(len(data)-i) {
				return male, female, false
			}
			i += int(l)
		case 5: // fixed32
			if i+4 > len(data) {
				return male, female, false
			}
			i += 4
		default:
			return male, female, false
		}
	}
	return male, female, true
}

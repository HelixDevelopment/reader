package mobi

import "fmt"

// decompressPalmDOC decodes the PalmDOC byte-oriented LZ77 representation.
// Each MOBI text record is independent; back-references may overlap newly
// decoded output, but may never refer before the beginning of the record.
func decompressPalmDOC(data []byte, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, fmt.Errorf("palmdoc: invalid output limit")
	}
	out := make([]byte, 0, min(len(data), 4096))
	ensure := func(additional int) error {
		if additional < 0 || int64(len(out))+int64(additional) > limit {
			return fmt.Errorf("palmdoc: decoded record exceeds %d byte limit", limit)
		}
		return nil
	}
	for i := 0; i < len(data); i++ {
		control := data[i]
		switch {
		case control == 0 || (control >= 0x09 && control <= 0x7f):
			if err := ensure(1); err != nil {
				return nil, err
			}
			out = append(out, control)
		case control >= 0x01 && control <= 0x08:
			count := int(control)
			if len(data)-i-1 < count {
				return nil, fmt.Errorf("palmdoc: truncated literal run")
			}
			if err := ensure(count); err != nil {
				return nil, err
			}
			out = append(out, data[i+1:i+1+count]...)
			i += count
		case control >= 0x80 && control <= 0xbf:
			if i+1 >= len(data) {
				return nil, fmt.Errorf("palmdoc: truncated back-reference")
			}
			i++
			pair := (uint16(control&0x3f) << 8) | uint16(data[i])
			distance := int(pair >> 3)
			length := int(pair&0x07) + 3
			if distance == 0 || distance > len(out) {
				return nil, fmt.Errorf("palmdoc: invalid back-reference distance %d", distance)
			}
			if err := ensure(length); err != nil {
				return nil, err
			}
			for range length {
				out = append(out, out[len(out)-distance])
			}
		default: // 0xc0..0xff: compressed space plus character.
			if err := ensure(2); err != nil {
				return nil, err
			}
			out = append(out, ' ', control^0x80)
		}
	}
	return out, nil
}

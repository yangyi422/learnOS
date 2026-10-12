package service

import (
	"bytes"
	"encoding/binary"
)

// Browsers apply JPEG EXIF orientation, whereas Go's decoder leaves pixels in
// storage order. Retain only that tag in renditions, never GPS/camera metadata.
func jpegOrientation(data []byte) uint16 {
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos < len(data); {
		if data[pos] != 0xff {
			return 1
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			return 1
		}
		marker := data[pos]
		pos++
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if pos+2 > len(data) {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		if size < 2 || size > len(data)-pos {
			return 1
		}
		segment := data[pos+2 : pos+size]
		pos += size
		if marker != 0xe1 || !bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			continue
		}
		tiff := segment[6:]
		if len(tiff) < 8 {
			continue
		}
		var order binary.ByteOrder
		switch string(tiff[:2]) {
		case "II":
			order = binary.LittleEndian
		case "MM":
			order = binary.BigEndian
		default:
			continue
		}
		if order.Uint16(tiff[2:4]) != 42 {
			continue
		}
		offset := uint64(order.Uint32(tiff[4:8]))
		if offset+2 > uint64(len(tiff)) {
			continue
		}
		count := uint64(order.Uint16(tiff[offset : offset+2]))
		offset += 2
		if offset+count*12 > uint64(len(tiff)) {
			continue
		}
		for i := uint64(0); i < count; i++ {
			entry := tiff[offset+i*12 : offset+(i+1)*12]
			if order.Uint16(entry[:2]) == 0x112 && order.Uint16(entry[2:4]) == 3 && order.Uint32(entry[4:8]) == 1 {
				value := order.Uint16(entry[8:10])
				if value >= 1 && value <= 8 {
					return value
				}
			}
		}
	}
	return 1
}

func retainJPEGOrientation(data []byte, orientation uint16) []byte {
	if orientation <= 1 || orientation > 8 {
		return data
	}
	// APP1 length 34: Exif signature + one little-endian TIFF SHORT entry.
	marker := []byte{0xff, 0xe1, 0, 34, 'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, byte(orientation), 0, 0, 0, 0, 0, 0, 0}
	out := make([]byte, 0, len(data)+len(marker))
	out = append(out, data[:2]...)
	out = append(out, marker...)
	return append(out, data[2:]...)
}

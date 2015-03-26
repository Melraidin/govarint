// TODO Can we skip storing the first bit of each value by assuming
// it's always a 1? Otherwise we'd have stored a 0 for the value's
// length.

package govarint

import (
	"fmt"
)

// Return the number of leading zeros before the first set bit.
func countLeadingZeros(x uint32) int {
	if x == 0 {
		return 32
	}

	count := 0
	if (x & 0xffff0000) == 0 {
		count += 16
		x = x << 16
	}
	if (x & 0xff000000) == 0 {
		count += 8
		x = x << 8
	}
	if (x & 0xf0000000) == 0 {
		count += 4
		x = x << 4
	}
	if (x & 0xc0000000) == 0 {
		count += 2
		x = x << 2
	}
	if (x & 0x80000000) == 0 {
		count += 1
	}
	return count
}

// TODO This should detect if there's additional bytes in the data
// after consuming all the fields specified and error out.
// func Decode(fields []uint8, data []byte) ([]uint32, error) {
// 	var dataBitIndex uint8
// 	var dataByteIndex int

// 	valueLengths := []uint8{}

// 	for _, fieldWidth := range fields {
// 		mask := uint16((1 << fieldWidth) - 1)
// 		mask <<= 8 - fieldWidth - dataBitIndex
// 		firstMask := uint8(mask) >> 8
// 		firstPart := firstMask & data[dataByteIndex]
// 		fmt.Printf("firstPart: %d\n", firstPart)

// 		secondPart := uint8(0)
// 		if fieldWidth > 8 {
// 			secondMask := uint8(mask)
// 			secondPart = secondMask & data[]
// 		}
// 	}

// 	return []uint32{}, nil
// }

/**
Encode the given values in the given varint format.

Args:
  fields: Ordered list of bit widths of fields. e.g.: 2 means two bits
    are allocated to specify the length of the value and so the value
    may only be in the range of ints expressible in two bits (0..3).
  values: Ordered list of values. If a value exceeds the allocated
    space an error will be returned.
*/
func Encode(fields []uint8, values []uint32) ([]byte, error) {
	if len(fields) != len(values) {
		return []byte{}, fmt.Errorf("mismatched field and value count, got %d fields and %d values", len(fields), len(values))
	}

	var formatCurByte uint8
	var formatCurIndex uint8
	var valueCurByte uint8
	var valueCurIndex uint8

	formatResult := []byte{}
	valueResult := []byte{}

	totalValueWidth := 0

	for i, fieldWidth := range fields {
		leadingZeros := countLeadingZeros(values[i])
		valueWidth := 32 - leadingZeros

		// Zero value, nothing to add to value byte.
		if valueWidth == 0 {
			addBitsToSlice(&formatResult, 0, fieldWidth, &formatCurByte, &formatCurIndex, false)

			continue
		}

		if uint64(values[i]) > uint64((1<<(1<<fieldWidth))-1) {
			return []byte{}, fmt.Errorf("value %d too large for field width %d", values[i], fieldWidth)
		}

		addBitsToSlice(&formatResult, uint32(valueWidth), fieldWidth, &formatCurByte, &formatCurIndex, false)

		addBitsToSlice(&valueResult, values[i], uint8(valueWidth), &valueCurByte, &valueCurIndex, true)

		totalValueWidth += valueWidth - 1
	}

	// Add trailing value bits.
	if valueCurIndex > 0 {
		addBitsToSlice(&valueResult, 0, 8-valueCurIndex, &valueCurByte, &valueCurIndex, false)
	}

	for _, b := range valueResult {
		curValueWidth := 8
		if totalValueWidth < 8 {
			curValueWidth = totalValueWidth
		}
		if curValueWidth == 8 {
			addBitsToSlice(&formatResult, uint32(b), uint8(curValueWidth), &formatCurByte, &formatCurIndex, false)
			totalValueWidth -= 8
		} else {
			addBitsToSlice(&formatResult, uint32(b>>uint(8-curValueWidth)), uint8(curValueWidth), &formatCurByte, &formatCurIndex, false)
		}
	}

	// Add trailing format bits.
	if formatCurIndex > 0 {
		addBitsToSlice(&formatResult, 0, 8-formatCurIndex, &formatCurByte, &formatCurIndex, false)
	}

	return formatResult, nil
}

func popBitsFromSlice(slice *[]byte, width uint8, curByte *uint8, curIndex *uint8, addFirstBit bool) (uint32, error) {
	// We only need to read from the current byte.
	if width+*curIndex <= 8 {
		mask := uint8((1 << width) - 1)
		mask <<= 8 - width - *curIndex

		fmt.Printf("mask: 0x%x, curByte&mask: 0x%x\n", mask, *curByte&mask)

		value := uint32((*curByte & mask) >> (8 - width - *curIndex))
		*curIndex += width

		if *curIndex < 8 {
			return value, nil
		}

		*curIndex = 0
		if len(*slice) != 0 {
			*curByte, *slice = (*slice)[0], (*slice)[1:len(*slice)]
		}

		return value, nil

		// firstMask := uint8(mask) >> 8
		// firstPart := firstMask & data[dataByteIndex]
		// fmt.Printf("firstPart: %d\n", firstPart)

		// secondPart := uint8(0)
		// if fieldWidth > 8 {
		// 	secondMask := uint8(mask)
		// 	secondPart = secondMask & data[]
		// }

	}

	return 0, fmt.Errorf("not yet implemented")
}

func addBitsToSlice(slice *[]byte, value uint32, width uint8, curByte *uint8, curIndex *uint8, skipFirstBit bool) {
	if skipFirstBit {
		width--
	}

	if width == 0 {
		return
	}

	// Shift value to high bits, stripping the first bit if skipFirstBit
	// since we know it must be set (no need to store).
	var v uint64 = (uint64(value) << uint(64-width-*curIndex))

	remainingBits := width

	start := uint(56)

	// Handle first partial byte
	if *curIndex > 0 {
		*curByte |= uint8((v & ^(0xff << (64 - *curIndex))) >> start)
		start -= 8

		completedWidth := 8 - *curIndex
		if width < completedWidth {
			completedWidth = width
		}

		if width+*curIndex >= 8 {
			*slice = append(*slice, *curByte)
			*curByte = 0
			*curIndex = 0
		} else {
			*curIndex += completedWidth
		}

		if completedWidth == width {
			return
		}

		remainingBits -= 8 - *curIndex
	}

	for i := start; remainingBits > 0; i -= 8 {
		*curIndex = remainingBits % 8

		*curByte = uint8(v >> i)
		if remainingBits < 8 {
			break
		}
		*slice = append(*slice, *curByte)
		remainingBits -= 8
	}
}

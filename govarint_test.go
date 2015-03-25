package govarint

import (
	"bytes"
	"testing"
)

type (
	leadingZeroTestCase struct {
		value uint32
		count int
	}
)

type encodeTestCase struct {
	fields []uint8
	values []uint32
	result []byte
}

type addBitsTestCase struct {
	slice         []byte
	value         uint32
	width         uint8
	curByte       uint8
	curIndex      uint8
	skipFirstBit  bool
	expectedSlice []byte
	expectedByte  uint8
	expectedIndex uint8
}

var (
	leadingZeroTests = []leadingZeroTestCase{
		{0, 32},
		{1, 31},
		{2, 30},
		{3, 30},
		{4, 29},
		{5, 29},
		{6, 29},
		{7, 29},
		{8, 28},
		{31, 27},
		{32, 26},
		{63, 26},
		{64, 25},
		{65, 25},
		{1<<32 - 1, 0},
	}

	encodeTests = []encodeTestCase{
		// Single zero value takes up only the space for the field
		// specifier.
		{[]uint8{1}, []uint32{0}, []byte{0}},
		{[]uint8{8}, []uint32{0}, []byte{0}},
		{[]uint8{9}, []uint32{0}, []byte{0, 0}},

		// // A one-value takes up only the space for the field specifier.
		{[]uint8{1}, []uint32{1}, []byte{0x80}},
		{[]uint8{2}, []uint32{1}, []byte{0x40}},

		// Single non-zero value takes up the space for the field
		// specifier plus len(value) - 1.
		{[]uint8{1}, []uint32{1}, []byte{0x80}},
		{[]uint8{2}, []uint32{3}, []byte{0xa0}},

		{[]uint8{2, 1}, []uint32{3, 0}, []byte{0x90}},
		{[]uint8{2, 1}, []uint32{3, 1}, []byte{0xb0}},
	}

	addBitsTests = []addBitsTestCase{
		{[]byte{}, 0, 0, 0, 0, false, []byte{}, 0, 0},

		// Single bit field with bit set should be only a 1 for field
		// specifier and nothing for value.
		{[]byte{}, 1, 1, 0, 0, false, []byte{}, 0x80, 1},

		{[]byte{}, 1 << 31, 32, 0, 0, false, []byte{0x80, 0, 0, 0}, 0, 0},

		{[]byte{0}, 2, 2, 0, 0, false, []byte{0}, 0x80, 2},
		{[]byte{0}, 2, 2, 0x80, 2, false, []byte{0}, 0xa0, 4},

		{[]byte{}, 1 << 31, 32, 0, 0, true, []byte{0, 0, 0}, 0, 7},
	}
)

func TestCountLeadingZeros(t *testing.T) {
	for _, tc := range leadingZeroTests {
		count := countLeadingZeros(tc.value)
		if tc.count != count {
			t.Errorf("Expected %d, got %d for %d", tc.count, count, tc.value)
		}
	}
}

func TestEncode(t *testing.T) {
	for _, tc := range encodeTests {
		result, err := Encode(tc.fields, tc.values)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if !bytes.Equal(result, tc.result) {
			t.Errorf("Expected 0x%x, got 0x%x for %v", tc.result, result, tc)
		}
	}
}

func TestInvalidEncode(t *testing.T) {
	_, err := Encode([]uint8{1}, []uint32{4})

	if err == nil {
		t.Errorf("Did not receive expected error")
	}

	expected := "value 4 too large for field width 1"
	if err.Error() != expected {
		t.Errorf("Expected error \"%s\", got: %s", expected, err)
	}
}

func TestAddBitsToSlice(t *testing.T) {
	for _, tc := range addBitsTests {
		addBitsToSlice(&tc.slice, tc.value, tc.width, &tc.curByte, &tc.curIndex, tc.skipFirstBit)

		if !bytes.Equal(tc.slice, tc.expectedSlice) {
			t.Errorf("Expected 0x%x, got 0x%x for %v", tc.expectedSlice, tc.slice, tc)
		}
		if tc.curByte != tc.expectedByte {
			t.Errorf("Expected current byte 0x%x, got 0x%x for %v", tc.expectedByte, tc.curByte, tc)
		}
		if tc.curIndex != tc.expectedIndex {
			t.Errorf("Expected current index %d, got %d for %v", tc.expectedIndex, tc.curIndex, tc)
		}
	}
}

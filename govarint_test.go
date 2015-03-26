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

type decodeTestCase struct {
	fields []uint8
	data   []byte
	values []uint32
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

type popBitsTestCase struct {
	slice         []byte
	width         uint8
	curByte       uint8
	curIndex      uint8
	addFirstBit   bool
	expectedValue uint32
	expectedError error
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
		{[]uint8{2}, []uint32{3}, []byte{0xa0}},

		{[]uint8{2, 1}, []uint32{3, 0}, []byte{0x90}},
		{[]uint8{2, 1}, []uint32{3, 1}, []byte{0xb0}},

		{[]uint8{2, 1}, []uint32{0, 0}, []byte{0}},
		{[]uint8{2, 1}, []uint32{0, 1}, []byte{0x20}},

		{[]uint8{2, 1}, []uint32{0, 1}, []byte{0x20}},

		{[]uint8{3, 1}, []uint32{0, 1}, []byte{0x10}},
		{[]uint8{3, 1}, []uint32{1, 1}, []byte{0x30}},
		{[]uint8{3, 1}, []uint32{2, 1}, []byte{0x50}},
		{[]uint8{3, 1}, []uint32{3, 1}, []byte{0x58}},
		{[]uint8{3, 1}, []uint32{4, 1}, []byte{0x70}},
		{[]uint8{3, 1}, []uint32{5, 1}, []byte{0x74}},
		{[]uint8{3, 1}, []uint32{6, 1}, []byte{0x78}},
		{[]uint8{3, 1}, []uint32{7, 1}, []byte{0x7c}},

		{[]uint8{3, 1}, []uint32{8, 1}, []byte{0x90}},
		{[]uint8{3, 1}, []uint32{9, 1}, []byte{0x92}},
		{[]uint8{3, 1}, []uint32{10, 1}, []byte{0x94}},
		{[]uint8{3, 1}, []uint32{11, 1}, []byte{0x96}},

		// TODO I think this should this end in 0xfc.
		{[]uint8{6, 1}, []uint32{0xffffffff, 1}, []byte{0x83, 0xff, 0xff, 0xfe}},
		// {[]uint8{6, 1}, []uint32{0xffffffff, 1}, []byte{0x83, 0xff, 0xff, 0xfc}},
	}

	// decodeTests = []decodeTestCase{
	// 	{[]uint8{1}, []byte{0}, []uint32{0}},
	// 	{[]uint8{8}, []byte{0}, []uint32{0}},
	// 	{[]uint8{9}, []byte{0, 0}, []uint32{0}},

	// 	{[]uint8{1}, []byte{0x80}, []uint32{1}},
	// 	{[]uint8{2}, []byte{0x40}, []uint32{1}},

	// 	{[]uint8{2}, []byte{0xa0}, []uint32{3}},
	// }

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

	popBitsTests = []popBitsTestCase{
		{[]byte{}, 1, 0x00, 0, false, 0, nil, []byte{}, 0x00, 1},
		{[]byte{}, 1, 0x80, 0, false, 1, nil, []byte{}, 0x80, 1},

		{[]byte{}, 1, 0x00, 1, false, 0, nil, []byte{}, 0x00, 2},
		{[]byte{}, 1, 0x40, 1, false, 1, nil, []byte{}, 0x40, 2},

		{[]byte{}, 1, 0x00, 7, false, 0, nil, []byte{}, 0x00, 0},
		{[]byte{}, 1, 0x01, 7, false, 1, nil, []byte{}, 0x01, 0},

		{[]byte{0xff}, 1, 0x00, 7, false, 0, nil, []byte{}, 0xff, 0},
		{[]byte{0xff}, 1, 0x01, 7, false, 1, nil, []byte{}, 0xff, 0},

		{[]byte{0x12, 0x34}, 1, 0x00, 7, false, 0, nil, []byte{0x34}, 0x12, 0},
		{[]byte{0x12, 0x34}, 1, 0x01, 7, false, 1, nil, []byte{0x34}, 0x12, 0},
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

// func TestDecode(t *testing.T) {
// 	for _, tc := range decodeTests {
// 		result, err := Decode(tc.fields, tc.data)
// 		if err != nil {
// 			t.Errorf("Unexpected error: %s", err)
// 		}

// 		if len(result) != len(tc.values) {
// 			t.Errorf("Expected %d values, got %d values for %v", len(tc.values), len(result), tc)
// 		}

// 		for i, expected := range tc.values {
// 			if expected != result[i] {
// 				t.Errorf("Expected 0x%x, got 0x%x for %v", expected, result[i], tc)
// 			}
// 		}
// 	}
// }

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

func TestPopBitsFromSlice(t *testing.T) {
	for _, tc := range popBitsTests {
		value, err := popBitsFromSlice(&tc.slice, tc.width, &tc.curByte, &tc.curIndex, tc.addFirstBit)

		if err != nil && tc.expectedError == nil {
			t.Errorf("Unexpected error %s for %v", err, tc)
		} else if err == nil && tc.expectedError != nil {
			t.Errorf("Expected error %s, received no error for %v", tc.expectedError, tc)
		} else if err != nil && tc.expectedError != nil && err.Error() != tc.expectedError.Error() {
			t.Errorf("Expected error %s, received error %s for %v", tc.expectedError, err, tc)
		}

		if value != tc.expectedValue {
			t.Errorf("Expected %d, got %d for %v", tc.expectedValue, value, tc)
		}
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

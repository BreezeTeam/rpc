package customer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type DataTest struct {
	I    interface{}
	P    *DataTest
	T    time.Duration
	A1   [4]string
	A2   [4]byte
	A3   [4]int
	A4   [4]float32
	S1   []string
	S2   []byte
	S3   []int
	S4   []float32
	M1   map[string]string
	M2   map[string]int
	M3   map[int]int
	M4   map[int]string
	M5   map[int]interface{}
	STR  string
	Int1 int64
	INT2 int8
	INT3 int16
	INT4 int32
	INT5 int64
	INT6 uint64
	INT7 uint32
	INT8 uint16
	INT9 uint8
	F1   float32
	F2   float64
}

var x = DataTest{
	//P:  &DataTest{I: 23},
	//I:  DataTest{I: 23},
	T:  time.Second * 3,
	A1: [4]string{"s1", "s2", "s2", "s2"},
	A2: [4]byte{'s', '2', '-', '.'},
	A3: [4]int{3, 2, 3, 13},
	A4: [4]float32{4.0, 43, 53, 6},
	S1: []string{"s1", "s2", "s2", "s2", ","},
	S2: []byte{'s', '2', '-', '.', ',', '-', '-', '-'},
	S3: []int{3, 2, 3, 13, 13, 14, 23},
	S4: []float32{4.0, 43, 53, 6, 76},
	M1: map[string]string{
		"M":  "1",
		"M2": "2",
	},
	M2: map[string]int{
		"M":  2,
		"M2": 3,
	},
	M3: map[int]int{
		1: 3,
		2: 4,
	},
	M4: map[int]string{
		2: "4",
		3: "5",
	},
	M5: map[int]interface{}{
		3: "1",
		4: "12",
	},
	STR:  "str",
	Int1: 11212121,
	INT2: 2,
	INT3: 3,
	INT4: 4,
	INT5: 5,
	INT6: 6,
	INT7: 7,
	INT8: 8,
	INT9: 9,
	F1:   1.0,
	F2:   0.0001,
}

type STest struct {
	T1 string
}

var x1 = STest{T1: "3213"}

func TestEncodeInterface(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	encoder := NewEncoder(buffer)
	if err := encoder.Encode(x1); err != nil {
		println(err.Error())
	}
	fmt.Printf("len: %d\n", buffer.Len())
	fmt.Printf("%v\n", buffer)

	println("decode")
	decoder := NewDecoder(buffer)
	var xx2 STest
	if err := decoder.Decode(&xx2); err != nil {
		println(err.Error())
	}
	fmt.Printf("%v\n", xx2)
}

func TestJsonEncoder(t *testing.T) {
	buffer2 := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buffer2)
	err := encoder.Encode(x)
	fmt.Printf("len: %d\n", buffer2.Len())
	fmt.Printf("%v\n", buffer2)
	println(err)
	json.NewDecoder(buffer2).Decode(&x)
	fmt.Printf("%v\n", x)
}

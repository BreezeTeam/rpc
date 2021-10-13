//通讯协议处理，主要处理封包和解包的过程
package customer

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type DataTest struct {
	T    time.Duration
	S1   []string
	S2   []byte
	S3   []int
	S4   []float32
	M1   map[string]string
	M2   map[string]int
	M3   map[int]int
	M4   map[int]string
	M5   map[int]interface{}
	I    interface{}
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

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(&src)
	if err != nil {
		return err
	}
	json.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
	return nil
}

func TestEncodeInterface(t *testing.T) {
	x := DataTest{
		T:  time.Second * 3,
		S1: []string{"s1", "s2", "s2", "s2", ","},
		S2: []byte{'s', '2', '-', '.', '-', '-', ',', '-', '-', '-'},
		S3: []int{3, 2, 3, 13, 13, 14, 23},
		S4: []float32{4.0, 43, 53, 6, 76},
		M1: map[string]string{
			"M": "1",
		},
		M2: map[string]int{
			"M": 2,
		},
		M3: map[int]int{
			1: 3,
		},
		M4: map[int]string{
			2: "4",
		},
		M5: map[int]interface{}{
			3: "1",
		},
		I:    nil,
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
	//copyx := &DataTest{}
	//err := DeepCopy(copyx, x)
	//if err != nil {
	//	println("error copying")
	//}
	//res, err := EncodeInterface(copyx)
	res, err := EncodeInterface(x)
	println(res)
	println(err)
}

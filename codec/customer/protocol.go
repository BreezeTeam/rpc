//通讯协议处理，主要处理封包和解包的过程
package customer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unsafe"
)

//字段长度
const (
	MagicNumber               = 0x6aed56e8
	ConstMagicNumber          = 4
	ConstProtocolLength       = 4
	ConstProtocolHeaderLength = 4
	ConstCodecType            = 4
	ConstConnectTimeout       = 8
	ConstHandleTimeout        = 8

	ConstSequence = 16

	ConstHeaderLength = 32
	ConstDataLength   = 4
)

// A Decoder reads and decodes JSON values from an input stream.
type Decoder struct {
	r       io.Reader
	buf     []byte
	scanp   int   // start of unread data in buf
	scanned int64 // amount of data already scanned
	err     error

	tokenState int
	tokenStack []int
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of JSON into a Go value.
func (dec *Decoder) Decode(v interface{}) error {
	if dec.err != nil {
		return dec.err
	}

	//if err := dec.tokenPrepareForDecode(); err != nil {
	//	return err
	//}
	//
	//if !dec.tokenValueAllowed() {
	//	return &SyntaxError{msg: "not at beginning of value", Offset: dec.InputOffset()}
	//}
	//
	//// Read whole value into buffer.
	//n, err := dec.readValue()
	//if err != nil {
	//	return err
	//}
	//dec.d.init(dec.buf[dec.scanp : dec.scanp+n])
	//dec.scanp += n
	//
	//// Don't save err from unmarshal into dec.err:
	//// the connection is still usable since we read a complete JSON
	//// object from it before the error happened.
	//err = dec.d.unmarshal(v)
	//
	//// fixup token streaming state
	//dec.tokenValueEnd()

	//return err
	return nil
}

// Buffered returns a reader of the data remaining in the Decoder's
// buffer. The reader is valid until the next call to Decode.
func (dec *Decoder) Buffered() io.Reader {
	return bytes.NewReader(dec.buf[dec.scanp:])
}

//func (dec *Decoder) readValue() (int, error) {
//	dec.scan.reset()
//
//	scanp := dec.scanp
//	var err error
//Input:
//	// help the compiler see that scanp is never negative, so it can remove
//	// some bounds checks below.
//	for scanp >= 0 {
//
//		// Look in the buffer for a new value.
//		for ; scanp < len(dec.buf); scanp++ {
//			c := dec.buf[scanp]
//			dec.scan.bytes++
//			switch dec.scan.step(&dec.scan, c) {
//			case scanEnd:
//				// scanEnd is delayed one byte so we decrement
//				// the scanner bytes count by 1 to ensure that
//				// this value is correct in the next call of Decode.
//				dec.scan.bytes--
//				break Input
//			case scanEndObject, scanEndArray:
//				// scanEnd is delayed one byte.
//				// We might block trying to get that byte from src,
//				// so instead invent a space byte.
//				if stateEndValue(&dec.scan, ' ') == scanEnd {
//					scanp++
//					break Input
//				}
//			case scanError:
//				dec.err = dec.scan.err
//				return 0, dec.scan.err
//			}
//		}
//
//		// Did the last read have an error?
//		// Delayed until now to allow buffer scan.
//		if err != nil {
//			if err == io.EOF {
//				if dec.scan.step(&dec.scan, ' ') == scanEnd {
//					break Input
//				}
//				if nonSpace(dec.buf) {
//					err = io.ErrUnexpectedEOF
//				}
//			}
//			dec.err = err
//			return 0, err
//		}
//
//		n := scanp - dec.scanp
//		err = dec.refill()
//		scanp = dec.scanp + n
//	}
//	return scanp - dec.scanp, nil
//}

func (dec *Decoder) refill() error {
	// Make room to read more into the buffer.
	// First slide down data already consumed.
	if dec.scanp > 0 {
		dec.scanned += int64(dec.scanp)
		n := copy(dec.buf, dec.buf[dec.scanp:])
		dec.buf = dec.buf[:n]
		dec.scanp = 0
	}

	// Grow buffer if not large enough.
	const minRead = 512
	if cap(dec.buf)-len(dec.buf) < minRead {
		newBuf := make([]byte, len(dec.buf), 2*cap(dec.buf)+minRead)
		copy(newBuf, dec.buf)
		dec.buf = newBuf
	}

	// Read. Delay error for next iteration (after scan).
	n, err := dec.r.Read(dec.buf[len(dec.buf):cap(dec.buf)])
	dec.buf = dec.buf[0 : len(dec.buf)+n]

	return err
}

// An Encoder writes JSON values to an output stream.
type Encoder struct {
	w   io.Writer
	err error

	indentBuf    *bytes.Buffer
	indentPrefix string
	indentValue  string
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes the JSON encoding of v to the stream,
// followed by a newline character.
//
// See the documentation for Marshal for details about the
// conversion of Go values to JSON.
func (enc *Encoder) Encode(v interface{}) error {
	if enc.err != nil {
		return enc.err
	}
	b, err := EncodeInterface(v)
	if err != nil {
		return err
	}
	if enc.indentPrefix != "" || enc.indentValue != "" {
		if enc.indentBuf == nil {
			enc.indentBuf = new(bytes.Buffer)
		}
		enc.indentBuf.Reset()
		//err = Indent(enc.indentBuf, b, enc.indentPrefix, enc.indentValue)
		if err != nil {

			return err
		}
		b = enc.indentBuf.Bytes()
	}

	//写入数据
	if _, err = enc.w.Write(b); err != nil {
		enc.err = err
	}

	return err
}

type EncoderError string

func (e EncoderError) Error() string {
	return string(e)
}

type EncoderItem struct {
	bytes []byte //字节序列
}

func EncodeWithBigEndian(buffer io.Writer, data interface{}) error {
	switch data.(type) {
	case *int:
		return EncoderError("cant encode *int")
	case int:
		return EncoderError("cant encode int")
	case []int:
		data = data.([]int64)
		return EncoderError("cant encode []int")
	case string:
		_, err := buffer.Write(String2Bytes(data.(string)))
		if err != nil {
			return EncoderError(err.Error())
		} else {
			return nil
		}
	default:
		err := binary.Write(buffer, binary.BigEndian, data)
		if err != nil {
			return EncoderError(err.Error())
		}
		return nil
	}
}

func numericalEncoder(buffer io.Writer, i interface{}) error {
	v := reflect.ValueOf(i)
	data := v.Interface()
	switch data.(type) {
	case *bool, bool, []bool, *int8, int8, []int8, *uint8, uint8, []uint8, *int16, int16, []int16, *uint16, uint16, []uint16, *int32, int32, []int32, *uint32, uint32, []uint32, *int64, int64, []int64, *uint64, uint64, []uint64, *float32, float32, []float32, *float64, float64, []float64:
		return EncodeWithBigEndian(buffer, data)
	case *int, int:
		return EncodeWithBigEndian(buffer, v.Int())
	case *uint, uint:
		return EncodeWithBigEndian(buffer, v.Uint())
	default:
		switch v.Kind() {
		case reflect.Int64:
			return EncodeWithBigEndian(buffer, data)
		default:
			return EncoderError("numericalEncoder Error")
		}
	}

}
func sliceEncoder(buffer io.Writer, i interface{}) error {
	iv := reflect.ValueOf(i)
	if iv.IsNil() {
		return nilEncoder(buffer, i)
	}
	data := iv.Interface()
	switch data.(type) {
	case []bool, []int8, []uint8, []int16, []uint16, []int32, []uint32, []int64, []uint64, []float32, []float64:
		return EncodeWithBigEndian(buffer, data)
	case []int, []uint:
		x := strconv.IntSize
		if n := x / 8; n != 0 {
			bs := make([]byte, n)
			switch v := data.(type) {
			case []int64:
				for i, x := range v {
					binary.BigEndian.PutUint64(bs[8*i:], uint64(x))
				}
			case []uint64:
				for i, x := range v {
					binary.BigEndian.PutUint64(bs[8*i:], x)
				}
			}
			_, err := buffer.Write(bs)
			return err
		}
		return nil
	default:
		n := iv.Len()
		err := EncodeWithBigEndian(buffer, "[")
		if err != nil {
			return err
		}
		for idx := 0; idx < n; idx++ {
			if idx > 0 {
				err = EncodeWithBigEndian(buffer, ",")
				if err != nil {
					return err
				}
			}
			if err = valueEncoder(buffer, iv.Index(idx).Interface()); err != nil {
				return err
			}
		}
		err = EncodeWithBigEndian(buffer, "]")
		if err != nil {
			return err
		}
		return nil
	}
}

func mapEncoder(buffer io.Writer, i interface{}) error {
	iv := reflect.ValueOf(i)
	if iv.IsNil() {
		return nilEncoder(buffer, i)
	}

	err := EncodeWithBigEndian(buffer, "{")
	if err != nil {
		return err
	}

	keys := iv.MapKeys()
	for i, k := range keys {
		if i > 0 {
			err = EncodeWithBigEndian(buffer, ",")
			if err != nil {
				return err
			}
		}
		m_key := k.Convert(iv.Type().Key())
		if err = valueEncoder(buffer, m_key.Interface()); err != nil {
			return err
		}
		err = EncodeWithBigEndian(buffer, ":")
		if err != nil {
			return err
		}
		if err = valueEncoder(buffer, iv.MapIndex(m_key).Interface()); err != nil {
			return err
		}

	}
	err = EncodeWithBigEndian(buffer, "}")
	if err != nil {
		return err
	}
	return nil
}

func stringEncoder(buffer io.Writer, i interface{}) error {
	return EncodeWithBigEndian(buffer, "\""+reflect.ValueOf(i).String()+"\"")
}
func invalidValueEncoder(buffer io.Writer, i interface{}) error {
	return EncodeWithBigEndian(buffer, "null")
}
func unsupportedTypeEncoder(buffer io.Writer, i interface{}) error {
	return EncoderError("UnsupportedTypeError")
}

func nilEncoder(buffer io.Writer, i interface{}) error {
	if reflect.ValueOf(i).IsNil() {
		return EncodeWithBigEndian(buffer, "null")
	}
	return EncoderError("must nil kind")
}

func interfaceEncoder(buffer io.Writer, i interface{}) error {
	if reflect.ValueOf(i).IsNil() {
		return nilEncoder(buffer, i)
	}
	return valueEncoder(buffer, i)
}

func structEncoder(buffer io.Writer, i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	it := reflect.TypeOf(i)

	if iv.Kind() != reflect.Struct {
		return EncoderError("need struct kind")
	}

	for i := 0; i < iv.NumField(); i++ {
		if err = valueEncoder(buffer, iv.Field(i).Interface()); err != nil {
			fmt.Printf("ERROR valueEncoder : name: %s, type: %s, value: %v\n", it.Field(i).Name, iv.Field(i).Type(), iv.Field(i).Interface())
			return err
		}
		fmt.Printf("valueEncoder : name: %s, type: %s, value: %v\n", it.Field(i).Name, iv.Field(i).Type(), iv.Field(i).Interface())
		fmt.Printf("buffer:%v\n", buffer)
	}
	return nil
}

func valueEncoder(buffer io.Writer, v interface{}) error {
	if !reflect.ValueOf(v).IsValid() {
		return invalidValueEncoder(buffer, v)
	}
	return newTypeEncoder(reflect.TypeOf(v), true)(buffer, v)
}

type encoderFunc func(w io.Writer, v interface{}) error

func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return numericalEncoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Struct:
		return structEncoder
	case reflect.Map:
		return mapEncoder
	case reflect.Slice:
		return sliceEncoder
	case reflect.Array:
		return sliceEncoder
	//case reflect.Ptr:
	//	return newPtrEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}

func EncodeInterface(v interface{}) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	err := valueEncoder(buffer, v)
	if err != nil {
		return nil, err
	}
	return []byte{}, nil
}

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
type RawMessage []byte

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

func IntToInt32ToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt32ToInt(b []byte) (x int) {
	var m int32
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &m)
	return int(m)
}

func Int32ToBytes(n int32) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt32(b []byte) (x int32) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}

func Int64ToBytes(n int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt64(b []byte) (x int64) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}
func Uint64ToBytes(n uint64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToUint64(b []byte) (x uint64) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}
func Int8ToBytes(n int8) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func BytesToInt8(b []byte) (x int8) {
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}

func String2Bytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func Bytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

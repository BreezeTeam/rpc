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
	ConstSequence             = 16
	ConstHeaderLength         = 32
	ConstDataLength           = 4
)

type Decoder struct {
	r       io.Reader
	buf     []byte
	scanp   int   //放置在未读字节首部
	scanned int64 // 已扫描部分
	err     error
	elemp   int //interface的地址指针
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

type DecoderError string

func (e DecoderError) Error() string {
	return string(e)
}
func (dec *Decoder) Decode(v interface{}) (err error) {
	if dec.err != nil {
		return dec.err
	}
	return dec.interfaceDecoder(v)
}
func (dec *Decoder) decodeWithBigEndian(buffer io.Writer, data interface{}) (err error) {
	switch data.(type) {
	case string:
		_, err = buffer.Write(String2Bytes(data.(string)))
		return err
	case []byte:
		_, err = buffer.Write(data.([]byte))
		return err
	default:
		err = binary.Read(bytes.NewBuffer(dec.buf), binary.BigEndian, &data)
		fmt.Printf("%v", data)
		fmt.Printf("%v", err)
		return err
	}
}

func (dec *Decoder) interfaceDecoder(v interface{}) (err error) {
	rv := reflect.ValueOf(v)
	//必须是指针类型
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		if reflect.TypeOf(v).Kind() != reflect.Ptr {
			return DecoderError("DecoderError: non-pointer,kind of (" + reflect.TypeOf(v).Kind().String() + ")")
		}
		return DecoderError("DecoderError: nil,kind of (" + reflect.TypeOf(v).Kind().String() + ")")
	}
	return dec.valueDecoder(v)
}

func (dec *Decoder) valueDecoder(v interface{}) (err error) {
	//以interface为模板,结合dec.buf中读取的数据进行数据解码
	// 1. 如果数据小于1个int,那就需要refill,将缓冲区装满
	if len(dec.buf)-dec.scanp < 8 {
		dec.refill()
	}
	//获取解码函数进行解码
	return dec.newTypeDecoder(reflect.TypeOf(v))(v)
}

type decoderFunc func(v interface{}) error

func (dec *Decoder) newTypeDecoder(t reflect.Type) decoderFunc {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return dec.numericalDecoder
	case reflect.String:
		return dec.stringDecoder
	case reflect.Interface:
		return dec.interfaceDecoder
	case reflect.Struct:
		return dec.structDecoder
	case reflect.Map:
		return dec.mapDecoder
	case reflect.Slice, reflect.Array:
		return dec.sliceDecoder
	case reflect.Ptr:
		return dec.ptrDecoder
	default:
		return dec.unsupportedTypeDecoder
	}
}
func (dec *Decoder) unsupportedTypeDecoder(i interface{}) error {
	return DecoderError("UnsupportedTypeError")
}

func (dec *Decoder) structDecoder(i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	if iv.Kind() != reflect.Struct {
		return DecoderError("need struct kind")
	}
	var lenx int64
	x := &lenx
	println(x)
	if err := dec.valueDecoder(&lenx); err != nil {
		err = fmt.Errorf(err.Error())
	}

	for i := 0; i < iv.NumField(); i++ {
		fmt.Printf("%v\n", iv.Kind())
		fmt.Printf("%v\n", iv.Field(i).Kind())
		if err = dec.valueDecoder(iv.Field(i).Interface()); err != nil {
			return err
		}
	}
	//if err = valueEncoder(buffer, structBuffer.Len()); err != nil {
	//	return err
	//}
	//return bytesEncoder(buffer, structBuffer.Bytes())
	return nil
}

func (dec *Decoder) numericalDecoder(i interface{}) error {
	buffer := bytes.NewBuffer([]byte{})
	v := reflect.ValueOf(i)
	data := v.Interface()
	switch data.(type) {
	case int64:
		return dec.decodeWithBigEndian(buffer, data.(int64))
	case *bool:
		return dec.decodeWithBigEndian(buffer, data.(*bool))
	case []bool:
		return dec.decodeWithBigEndian(buffer, data.([]bool))
	case *int8:
		return dec.decodeWithBigEndian(buffer, data.(*int8))
	case []int8:
		return dec.decodeWithBigEndian(buffer, data.([]int8))
	case *uint8:
		return dec.decodeWithBigEndian(buffer, data.(*uint8))
	case []uint8:
		return dec.decodeWithBigEndian(buffer, data.([]uint8))
	case *int16:
		return dec.decodeWithBigEndian(buffer, data.(*int16))
	case []int16:
		return dec.decodeWithBigEndian(buffer, data.([]int16))
	case *int32:
		return dec.decodeWithBigEndian(buffer, data.(*int32))
	case []int32:
		return dec.decodeWithBigEndian(buffer, data.([]int32))
	case *uint32:
		return dec.decodeWithBigEndian(buffer, data.(*uint32))
	case []uint32:
		return dec.decodeWithBigEndian(buffer, data.([]uint32))
	case *int64:
		return dec.decodeWithBigEndian(buffer, data.(*int64))
	case []int64:
		return dec.decodeWithBigEndian(buffer, data.([]int64))
	case *uint64:
		return dec.decodeWithBigEndian(buffer, data.(*uint64))
	case []uint64:
		return dec.decodeWithBigEndian(buffer, data.([]uint64))
	case *float32:
		return dec.decodeWithBigEndian(buffer, data.(*float32))
	case []float32:
		return dec.decodeWithBigEndian(buffer, data.([]float32))
	case *float64:
		return dec.decodeWithBigEndian(buffer, data.(*float64))
	case []float64:
		return dec.decodeWithBigEndian(buffer, data.([]float64))
	case interface{}:
		return dec.decodeWithBigEndian(buffer, data)
	default:
		return dec.decodeWithBigEndian(buffer, data)
	}
	return nil
}
func (dec *Decoder) sliceDecoder(i interface{}) (err error) {
	//iv := reflect.ValueOf(i)
	////写入KV对的数量
	//if err = encodeWithBigEndian(buffer, iv.Len()); err != nil || iv.Len() == 0 {
	//	return err
	//}
	//data := iv.Interface()
	//switch data.(type) {
	//case []bool, []int8, []uint8, []int16, []uint16, []int32, []uint32, []int64, []uint64, []float32, []float64:
	//	return encodeWithBigEndian(buffer, data)
	//case []int, []uint:
	//	x := strconv.IntSize
	//	if n := x / 8; n != 0 {
	//		bs := make([]byte, n)
	//		switch v := data.(type) {
	//		case []int64:
	//			for i, x := range v {
	//				binary.BigEndian.PutUint64(bs[8*i:], uint64(x))
	//			}
	//		case []uint64:
	//			for i, x := range v {
	//				binary.BigEndian.PutUint64(bs[8*i:], x)
	//			}
	//		}
	//		_, err = buffer.Write(bs)
	//		return err
	//	}
	//	return err
	//default:
	//	for idx := 0; idx < iv.Len(); idx++ {
	//		if err = valueEncoder(buffer, iv.Index(idx).Interface()); err != nil {
	//			return err
	//		}
	//	}
	//	return nil
	//}
	return nil
}

func (dec *Decoder) mapDecoder(i interface{}) (err error) {
	//iv := reflect.ValueOf(i)
	//keys := iv.MapKeys()
	////写入KV对的数量
	//if err = encodeWithBigEndian(buffer, len(keys)); err != nil || len(keys) == 0 {
	//	return err
	//}
	//for _, k := range keys {
	//	m_key := k.Convert(iv.Type().Key())
	//	if err = valueEncoder(buffer, m_key.Interface()); err != nil {
	//		return err
	//	}
	//	if err = valueEncoder(buffer, iv.MapIndex(m_key).Interface()); err != nil {
	//		return err
	//	}
	//}
	//return err
	return nil
}
func (dec *Decoder) bytesDecoder(v interface{}) error {
	return nil
	//return encodeWithBigEndian(buffer, v)
}

func (dec *Decoder) stringDecoder(i interface{}) (err error) {
	var len int64
	if err := dec.valueDecoder(&len); err != nil {
		err = fmt.Errorf(err.Error())
	}

	//structBuffer := bytes.NewBuffer([]byte{})
	//if err = encodeWithBigEndian(structBuffer, reflect.ValueOf(i).String()); err != nil {
	//	return err
	//}
	//if err = valueEncoder(buffer, structBuffer.Len()); err != nil {
	//	return err
	//}
	//return bytesEncoder(buffer, structBuffer.Bytes())
	return nil
}

func (dec *Decoder) invalidValuedEcoder(i interface{}) error {
	//return encodeWithBigEndian(buffer, 0)
	return nil
}

func (dec *Decoder) unsupportedTypedDecoder(i interface{}) error {
	return EncoderError("UnsupportedTypeError")
}

func (dec *Decoder) ptrDecoder(i interface{}) error {
	iv := reflect.ValueOf(i)
	if iv.IsNil() {
		return dec.nilDecoder(i)
	}
	return dec.valueDecoder(iv.Elem().Interface())
}

func (dec *Decoder) nilDecoder(i interface{}) error {
	//if reflect.ValueOf(i).IsNil() {
	//	return encodeWithBigEndian(buffer, uint8(0))
	//}
	return EncoderError("must nil kind")
}

func (dec *Decoder) refill() error {
	if dec.scanp > 0 {
		dec.scanned += int64(dec.scanp)
		n := copy(dec.buf, dec.buf[dec.scanp:])
		dec.buf = dec.buf[:n]
		dec.scanp = 0
	}
	const minRead = 512
	if cap(dec.buf)-len(dec.buf) < minRead {
		newBuf := make([]byte, len(dec.buf), 2*cap(dec.buf)+minRead)
		copy(newBuf, dec.buf)
		dec.buf = newBuf
	}
	n, err := dec.r.Read(dec.buf[len(dec.buf):cap(dec.buf)])
	dec.buf = dec.buf[0 : len(dec.buf)+n]
	return err
}

type Encoder struct {
	w   io.Writer
	err error
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (enc *Encoder) Encode(v interface{}) error {
	if enc.err != nil {
		return enc.err
	}
	//编码Interface
	b, err := enc.encodeInterface(v)
	if err != nil {
		return err
	}
	//写入数据
	if _, err = enc.w.Write(b); err != nil {
		enc.err = err
	}

	return err
}

func (enc *Encoder) encodeInterface(v interface{}) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	err := valueEncoder(buffer, v)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type EncoderError string

func (e EncoderError) Error() string {
	return string(e)
}

func encodeWithBigEndian(buffer io.Writer, data interface{}) (err error) {
	switch data.(type) {
	case *int, int:
		v := reflect.ValueOf(data)
		return encodeWithBigEndian(buffer, v.Int())
	case *uint, uint:
		v := reflect.ValueOf(data)
		return encodeWithBigEndian(buffer, v.Uint())
	case string:
		_, err = buffer.Write(String2Bytes(data.(string)))
		return err
	case []byte:
		_, err = buffer.Write(data.([]byte))
		return err
	default:
		return binary.Write(buffer, binary.BigEndian, data)
	}
}

func numericalEncoder(buffer io.Writer, i interface{}) error {
	v := reflect.ValueOf(i)
	data := v.Interface()
	switch data.(type) {
	case *bool, bool, []bool, *int8, int8, []int8, *uint8, uint8, []uint8, *int16, int16, []int16, *uint16, uint16, []uint16, *int32, int32, []int32, *uint32, uint32, []uint32, *int64, int64, []int64, *uint64, uint64, []uint64, *float32, float32, []float32, *float64, float64, []float64:
		return encodeWithBigEndian(buffer, data)
	case *int, int:
		return encodeWithBigEndian(buffer, v.Int())
	case *uint, uint:
		return encodeWithBigEndian(buffer, v.Uint())
	default:
		return encodeWithBigEndian(buffer, data)
	}
}
func sliceEncoder(buffer io.Writer, i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	//写入KV对的数量
	if err = encodeWithBigEndian(buffer, iv.Len()); err != nil || iv.Len() == 0 {
		return err
	}
	data := iv.Interface()
	switch data.(type) {
	case []bool, []int8, []uint8, []int16, []uint16, []int32, []uint32, []int64, []uint64, []float32, []float64:
		return encodeWithBigEndian(buffer, data)
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
			_, err = buffer.Write(bs)
			return err
		}
		return err
	default:
		for idx := 0; idx < iv.Len(); idx++ {
			if err = valueEncoder(buffer, iv.Index(idx).Interface()); err != nil {
				return err
			}
		}
		return nil
	}
}

func mapEncoder(buffer io.Writer, i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	keys := iv.MapKeys()
	//写入KV对的数量
	if err = encodeWithBigEndian(buffer, len(keys)); err != nil || len(keys) == 0 {
		return err
	}
	for _, k := range keys {
		m_key := k.Convert(iv.Type().Key())
		if err = valueEncoder(buffer, m_key.Interface()); err != nil {
			return err
		}
		if err = valueEncoder(buffer, iv.MapIndex(m_key).Interface()); err != nil {
			return err
		}
	}
	return err
}
func bytesEncoder(buffer io.Writer, v interface{}) error {
	return encodeWithBigEndian(buffer, v)
}

func stringEncoder(buffer io.Writer, i interface{}) (err error) {
	structBuffer := bytes.NewBuffer([]byte{})
	if err = encodeWithBigEndian(structBuffer, reflect.ValueOf(i).String()); err != nil {
		return err
	}
	if err = valueEncoder(buffer, structBuffer.Len()); err != nil {
		return err
	}
	return bytesEncoder(buffer, structBuffer.Bytes())
}

func invalidValueEncoder(buffer io.Writer, i interface{}) error {
	return encodeWithBigEndian(buffer, 0)
}

func unsupportedTypeEncoder(buffer io.Writer, i interface{}) error {
	return EncoderError("UnsupportedTypeError")
}

func ptrEncoder(buffer io.Writer, i interface{}) error {
	iv := reflect.ValueOf(i)
	if iv.IsNil() {
		return nilEncoder(buffer, i)
	}
	return valueEncoder(buffer, iv.Elem().Interface())
}

func nilEncoder(buffer io.Writer, i interface{}) error {
	if reflect.ValueOf(i).IsNil() {
		return encodeWithBigEndian(buffer, uint8(0))
	}
	return EncoderError("must nil kind")
}

func interfaceEncoder(buffer io.Writer, i interface{}) error {
	return valueEncoder(buffer, i)
}

func structEncoder(buffer io.Writer, i interface{}) (err error) {
	iv := reflect.ValueOf(i)
	if iv.Kind() != reflect.Struct {
		return EncoderError("need struct kind")
	}
	structBuffer := bytes.NewBuffer([]byte{})
	for i := 0; i < iv.NumField(); i++ {
		if err = valueEncoder(structBuffer, iv.Field(i).Interface()); err != nil {
			return err
		}
	}
	if err = valueEncoder(buffer, structBuffer.Len()); err != nil {
		return err
	}
	return bytesEncoder(buffer, structBuffer.Bytes())
}

func valueEncoder(buffer io.Writer, v interface{}) error {
	if !reflect.ValueOf(v).IsValid() || v == nil {
		return invalidValueEncoder(buffer, v)
	}
	return newTypeEncoder(reflect.TypeOf(v))(buffer, v)
}

type encoderFunc func(w io.Writer, v interface{}) error

func newTypeEncoder(t reflect.Type) encoderFunc {
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
	case reflect.Slice, reflect.Array:
		return sliceEncoder
	case reflect.Ptr:
		return ptrEncoder
	default:
		return unsupportedTypeEncoder
	}
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

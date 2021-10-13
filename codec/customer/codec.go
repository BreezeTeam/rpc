package customer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"rpc/codec"
	"rpc/metadata"
)

type jsonCodec struct {
	buf *bytes.Buffer
	mt  metadata.MessageType
	rwc io.ReadWriteCloser
	c   *clientCodec
	s   *serverCodec
}

func (j *jsonCodec) Close() error {
	j.buf.Reset()
	return j.rwc.Close()
}

func (j *jsonCodec) String() string {
	return "json-rpc"
}

func (j *jsonCodec) Write(m *metadata.Message, b interface{}) error {
	switch m.Type {
	case metadata.Request:
		return j.c.Write(m, b)
	case metadata.Response, metadata.Error:
		return j.s.Write(m, b)
	case metadata.Event:
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		_, err = j.rwc.Write(data)
		return err
	default:
		return fmt.Errorf("Unrecognised message type: %v", m.Type)
	}
}

func (j *jsonCodec) ReadHeader(m *metadata.Message, mt metadata.MessageType) error {
	j.buf.Reset()
	j.mt = mt

	switch mt {
	case metadata.Request:
		return j.s.ReadHeader(m)
	case metadata.Response:
		return j.c.ReadHeader(m)
	case metadata.Event:
		_, err := io.Copy(j.buf, j.rwc)
		return err
	default:
		return fmt.Errorf("Unrecognised message type: %v", mt)
	}
}

func (j *jsonCodec) ReadBody(b interface{}) error {
	switch j.mt {
	case metadata.Request:
		return j.s.ReadBody(b)
	case metadata.Response:
		return j.c.ReadBody(b)
	case metadata.Event:
		if b != nil {
			return json.Unmarshal(j.buf.Bytes(), b)
		}
	default:
		return fmt.Errorf("Unrecognised message type: %v", j.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &jsonCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
		c:   newClientCodec(rwc),
		s:   newServerCodec(rwc),
	}
}

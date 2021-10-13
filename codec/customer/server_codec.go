package customer

import (
	"encoding/json"
	"fmt"
	"io"
	"rpc/metadata"
)

type serverCodec struct {
	dec *json.Decoder // for reading 0x6aed56e8 values
	enc *json.Encoder // for writing 0x6aed56e8 values
	c   io.Closer

	// temporary work space
	req  serverRequest
	resp serverResponse
}

type serverRequest struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	ID     interface{}      `json:"id"`
}

type serverResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func (r *serverRequest) reset() {
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[0:0]
	}
	if r.ID != nil {
		r.ID = nil
	}
}

func newServerCodec(conn io.ReadWriteCloser) *serverCodec {
	return &serverCodec{
		dec: json.NewDecoder(conn),
		enc: json.NewEncoder(conn),
		c:   conn,
	}
}

func (c *serverCodec) ReadHeader(m *metadata.Message) error {
	c.req.reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	m.Method = c.req.Method
	m.Id = fmt.Sprintf("%v", c.req.ID)
	c.req.ID = nil
	return nil
}

func (c *serverCodec) ReadBody(x interface{}) error {
	if x == nil {
		return nil
	}
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.req.Params, &params)
}

var null = json.RawMessage([]byte("null"))

func (c *serverCodec) Write(m *metadata.Message, x interface{}) error {
	var resp serverResponse
	resp.ID = m.Id
	resp.Result = x
	if m.Error == "" {
		resp.Error = nil
	} else {
		resp.Error = m.Error
	}
	return c.enc.Encode(resp)
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}

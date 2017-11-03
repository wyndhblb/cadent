package metrics

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *GraphiteApiItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zbai uint32
	zbai, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zbai) {
		(*z) = (*z)[:zbai]
	} else {
		(*z) = make(GraphiteApiItems, zbai)
	}
	for zbzg := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zbzg] = nil
		} else {
			if (*z)[zbzg] == nil {
				(*z)[zbzg] = new(GraphiteApiItem)
			}
			err = (*z)[zbzg].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z GraphiteApiItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zcmr := range z {
		if z[zcmr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zcmr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z GraphiteApiItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zcmr := range z {
		if z[zcmr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zcmr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *GraphiteApiItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zwht uint32
	zwht, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zwht) {
		(*z) = (*z)[:zwht]
	} else {
		(*z) = make(GraphiteApiItems, zwht)
	}
	for zajw := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zajw] = nil
		} else {
			if (*z)[zajw] == nil {
				(*z)[zajw] = new(GraphiteApiItem)
			}
			bts, err = (*z)[zajw].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z GraphiteApiItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zhct := range z {
		if z[zhct] == nil {
			s += msgp.NilSize
		} else {
			s += z[zhct].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawDataPointList) DecodeMsg(dc *msgp.Reader) (err error) {
	var zlqf uint32
	zlqf, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zlqf) {
		(*z) = (*z)[:zlqf]
	} else {
		(*z) = make(RawDataPointList, zlqf)
	}
	for zxhx := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zxhx] = nil
		} else {
			if (*z)[zxhx] == nil {
				(*z)[zxhx] = new(RawDataPoint)
			}
			err = (*z)[zxhx].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawDataPointList) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zdaf := range z {
		if z[zdaf] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zdaf].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawDataPointList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zdaf := range z {
		if z[zdaf] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zdaf].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawDataPointList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zjfb uint32
	zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zjfb) {
		(*z) = (*z)[:zjfb]
	} else {
		(*z) = make(RawDataPointList, zjfb)
	}
	for zpks := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zpks] = nil
		} else {
			if (*z)[zpks] == nil {
				(*z)[zpks] = new(RawDataPoint)
			}
			bts, err = (*z)[zpks].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawDataPointList) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zcxo := range z {
		if z[zcxo] == nil {
			s += msgp.NilSize
		} else {
			s += z[zcxo].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawRenderItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zxpk uint32
	zxpk, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zxpk) {
		(*z) = (*z)[:zxpk]
	} else {
		(*z) = make(RawRenderItems, zxpk)
	}
	for zrsw := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zrsw] = nil
		} else {
			if (*z)[zrsw] == nil {
				(*z)[zrsw] = new(RawRenderItem)
			}
			err = (*z)[zrsw].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawRenderItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zdnj := range z {
		if z[zdnj] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zdnj].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawRenderItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zdnj := range z {
		if z[zdnj] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zdnj].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawRenderItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zsnv uint32
	zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zsnv) {
		(*z) = (*z)[:zsnv]
	} else {
		(*z) = make(RawRenderItems, zsnv)
	}
	for zobc := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zobc] = nil
		} else {
			if (*z)[zobc] == nil {
				(*z)[zobc] = new(RawRenderItem)
			}
			bts, err = (*z)[zobc].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawRenderItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zkgt := range z {
		if z[zkgt] == nil {
			s += msgp.NilSize
		} else {
			s += z[zkgt].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RenderItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zqke uint32
	zqke, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zqke) {
		(*z) = (*z)[:zqke]
	} else {
		(*z) = make(RenderItems, zqke)
	}
	for zpez := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zpez] = nil
		} else {
			if (*z)[zpez] == nil {
				(*z)[zpez] = new(RenderItem)
			}
			err = (*z)[zpez].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RenderItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zqyh := range z {
		if z[zqyh] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zqyh].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RenderItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zqyh := range z {
		if z[zqyh] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zqyh].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RenderItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zywj uint32
	zywj, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zywj) {
		(*z) = (*z)[:zywj]
	} else {
		(*z) = make(RenderItems, zywj)
	}
	for zyzr := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zyzr] = nil
		} else {
			if (*z)[zyzr] == nil {
				(*z)[zyzr] = new(RenderItem)
			}
			bts, err = (*z)[zyzr].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RenderItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zjpj := range z {
		if z[zjpj] == nil {
			s += msgp.NilSize
		} else {
			s += z[zjpj].Msgsize()
		}
	}
	return
}

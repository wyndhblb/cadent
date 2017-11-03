package repr

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *NilJsonFloat64) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zxvk float64
		zxvk, err = dc.ReadFloat64()
		(*z) = NilJsonFloat64(zxvk)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z NilJsonFloat64) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteFloat64(float64(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z NilJsonFloat64) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendFloat64(o, float64(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *NilJsonFloat64) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zbzg float64
		zbzg, bts, err = msgp.ReadFloat64Bytes(bts)
		(*z) = NilJsonFloat64(zbzg)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z NilJsonFloat64) Msgsize() (s int) {
	s = msgp.Float64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *ReprList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zcmr uint32
	zcmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "MinTime":
			z.MinTime, err = dc.ReadTime()
			if err != nil {
				return
			}
		case "MaxTime":
			z.MaxTime, err = dc.ReadTime()
			if err != nil {
				return
			}
		case "Reprs":
			var zajw uint32
			zajw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Reprs) >= int(zajw) {
				z.Reprs = (z.Reprs)[:zajw]
			} else {
				z.Reprs = make([]StatRepr, zajw)
			}
			for zbai := range z.Reprs {
				err = z.Reprs[zbai].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *ReprList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "MinTime"
	err = en.Append(0x83, 0xa7, 0x4d, 0x69, 0x6e, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteTime(z.MinTime)
	if err != nil {
		return
	}
	// write "MaxTime"
	err = en.Append(0xa7, 0x4d, 0x61, 0x78, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteTime(z.MaxTime)
	if err != nil {
		return
	}
	// write "Reprs"
	err = en.Append(0xa5, 0x52, 0x65, 0x70, 0x72, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Reprs)))
	if err != nil {
		return
	}
	for zbai := range z.Reprs {
		err = z.Reprs[zbai].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *ReprList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "MinTime"
	o = append(o, 0x83, 0xa7, 0x4d, 0x69, 0x6e, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendTime(o, z.MinTime)
	// string "MaxTime"
	o = append(o, 0xa7, 0x4d, 0x61, 0x78, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendTime(o, z.MaxTime)
	// string "Reprs"
	o = append(o, 0xa5, 0x52, 0x65, 0x70, 0x72, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Reprs)))
	for zbai := range z.Reprs {
		o, err = z.Reprs[zbai].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ReprList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zwht uint32
	zwht, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zwht > 0 {
		zwht--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "MinTime":
			z.MinTime, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				return
			}
		case "MaxTime":
			z.MaxTime, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				return
			}
		case "Reprs":
			var zhct uint32
			zhct, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Reprs) >= int(zhct) {
				z.Reprs = (z.Reprs)[:zhct]
			} else {
				z.Reprs = make([]StatRepr, zhct)
			}
			for zbai := range z.Reprs {
				bts, err = z.Reprs[zbai].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ReprList) Msgsize() (s int) {
	s = 1 + 8 + msgp.TimeSize + 8 + msgp.TimeSize + 6 + msgp.ArrayHeaderSize
	for zbai := range z.Reprs {
		s += z.Reprs[zbai].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatId) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zcua uint64
		zcua, err = dc.ReadUint64()
		(*z) = StatId(zcua)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z StatId) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteUint64(uint64(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z StatId) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendUint64(o, uint64(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatId) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zxhx uint64
		zxhx, bts, err = msgp.ReadUint64Bytes(bts)
		(*z) = StatId(zxhx)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z StatId) Msgsize() (s int) {
	s = msgp.Uint64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatNameSlice) DecodeMsg(dc *msgp.Reader) (err error) {
	var zpks uint32
	zpks, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zpks) {
		(*z) = (*z)[:zpks]
	} else {
		(*z) = make(StatNameSlice, zpks)
	}
	for zdaf := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zdaf] = nil
		} else {
			if (*z)[zdaf] == nil {
				(*z)[zdaf] = new(StatName)
			}
			err = (*z)[zdaf].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z StatNameSlice) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zjfb := range z {
		if z[zjfb] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zjfb].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z StatNameSlice) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zjfb := range z {
		if z[zjfb] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zjfb].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatNameSlice) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zeff uint32
	zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zeff) {
		(*z) = (*z)[:zeff]
	} else {
		(*z) = make(StatNameSlice, zeff)
	}
	for zcxo := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zcxo] = nil
		} else {
			if (*z)[zcxo] == nil {
				(*z)[zcxo] = new(StatName)
			}
			bts, err = (*z)[zcxo].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z StatNameSlice) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zrsw := range z {
		if z[zrsw] == nil {
			s += msgp.NilSize
		} else {
			s += z[zrsw].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatReprSlice) DecodeMsg(dc *msgp.Reader) (err error) {
	var zobc uint32
	zobc, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zobc) {
		(*z) = (*z)[:zobc]
	} else {
		(*z) = make(StatReprSlice, zobc)
	}
	for zdnj := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zdnj] = nil
		} else {
			if (*z)[zdnj] == nil {
				(*z)[zdnj] = new(StatRepr)
			}
			err = (*z)[zdnj].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z StatReprSlice) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zsnv := range z {
		if z[zsnv] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zsnv].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z StatReprSlice) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zsnv := range z {
		if z[zsnv] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zsnv].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatReprSlice) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zema uint32
	zema, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zema) {
		(*z) = (*z)[:zema]
	} else {
		(*z) = make(StatReprSlice, zema)
	}
	for zkgt := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zkgt] = nil
		} else {
			if (*z)[zkgt] == nil {
				(*z)[zkgt] = new(StatRepr)
			}
			bts, err = (*z)[zkgt].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z StatReprSlice) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zpez := range z {
		if z[zpez] == nil {
			s += msgp.NilSize
		} else {
			s += z[zpez].Msgsize()
		}
	}
	return
}

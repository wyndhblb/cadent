package indexer

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *MetricFindItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zbai uint32
	zbai, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zbai) {
		(*z) = (*z)[:zbai]
	} else {
		(*z) = make(MetricFindItems, zbai)
	}
	for zbzg := range *z {
		err = (*z)[zbzg].DecodeMsg(dc)
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z MetricFindItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zcmr := range z {
		err = z[zcmr].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricFindItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zcmr := range z {
		o, err = z[zcmr].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricFindItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zwht uint32
	zwht, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zwht) {
		(*z) = (*z)[:zwht]
	} else {
		(*z) = make(MetricFindItems, zwht)
	}
	for zajw := range *z {
		bts, err = (*z)[zajw].UnmarshalMsg(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z MetricFindItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zhct := range z {
		s += z[zhct].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricListItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zlqf uint32
	zlqf, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zlqf) {
		(*z) = (*z)[:zlqf]
	} else {
		(*z) = make(MetricListItems, zlqf)
	}
	for zxhx := range *z {
		(*z)[zxhx], err = dc.ReadString()
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z MetricListItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zdaf := range z {
		err = en.WriteString(z[zdaf])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricListItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zdaf := range z {
		o = msgp.AppendString(o, z[zdaf])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricListItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zjfb uint32
	zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zjfb) {
		(*z) = (*z)[:zjfb]
	} else {
		(*z) = make(MetricListItems, zjfb)
	}
	for zpks := range *z {
		(*z)[zpks], bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z MetricListItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zcxo := range z {
		s += msgp.StringPrefixSize + len(z[zcxo])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricTagItems) DecodeMsg(dc *msgp.Reader) (err error) {
	var zxpk uint32
	zxpk, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zxpk) {
		(*z) = (*z)[:zxpk]
	} else {
		(*z) = make(MetricTagItems, zxpk)
	}
	for zrsw := range *z {
		err = (*z)[zrsw].DecodeMsg(dc)
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z MetricTagItems) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zdnj := range z {
		err = z[zdnj].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricTagItems) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zdnj := range z {
		o, err = z[zdnj].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricTagItems) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zsnv uint32
	zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zsnv) {
		(*z) = (*z)[:zsnv]
	} else {
		(*z) = make(MetricTagItems, zsnv)
	}
	for zobc := range *z {
		bts, err = (*z)[zobc].UnmarshalMsg(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z MetricTagItems) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zkgt := range z {
		s += z[zkgt].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TagNameList) DecodeMsg(dc *msgp.Reader) (err error) {
	var zqke uint32
	zqke, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zqke) {
		(*z) = (*z)[:zqke]
	} else {
		(*z) = make(TagNameList, zqke)
	}
	for zpez := range *z {
		(*z)[zpez], err = dc.ReadString()
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z TagNameList) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zqyh := range z {
		err = en.WriteString(z[zqyh])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z TagNameList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zqyh := range z {
		o = msgp.AppendString(o, z[zqyh])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TagNameList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zywj uint32
	zywj, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zywj) {
		(*z) = (*z)[:zywj]
	} else {
		(*z) = make(TagNameList, zywj)
	}
	for zyzr := range *z {
		(*z)[zyzr], bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z TagNameList) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zjpj := range z {
		s += msgp.StringPrefixSize + len(z[zjpj])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UidList) DecodeMsg(dc *msgp.Reader) (err error) {
	var zgmo uint32
	zgmo, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zgmo) {
		(*z) = (*z)[:zgmo]
	} else {
		(*z) = make(UidList, zgmo)
	}
	for zrfe := range *z {
		(*z)[zrfe], err = dc.ReadString()
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z UidList) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for ztaf := range z {
		err = en.WriteString(z[ztaf])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z UidList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for ztaf := range z {
		o = msgp.AppendString(o, z[ztaf])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UidList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zsbz uint32
	zsbz, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zsbz) {
		(*z) = (*z)[:zsbz]
	} else {
		(*z) = make(UidList, zsbz)
	}
	for zeth := range *z {
		(*z)[zeth], bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z UidList) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zrjx := range z {
		s += msgp.StringPrefixSize + len(z[zrjx])
	}
	return
}

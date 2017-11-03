package indexer

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/schemas/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *MetricExpandItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zbzg uint32
	zbzg, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zbzg > 0 {
		zbzg--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "results":
			var zbai uint32
			zbai, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Results) >= int(zbai) {
				z.Results = (z.Results)[:zbai]
			} else {
				z.Results = make([]string, zbai)
			}
			for zxvk := range z.Results {
				z.Results[zxvk], err = dc.ReadString()
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
func (z *MetricExpandItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "results"
	err = en.Append(0x81, 0xa7, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Results)))
	if err != nil {
		return
	}
	for zxvk := range z.Results {
		err = en.WriteString(z.Results[zxvk])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricExpandItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "results"
	o = append(o, 0x81, 0xa7, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Results)))
	for zxvk := range z.Results {
		o = msgp.AppendString(o, z.Results[zxvk])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricExpandItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcmr uint32
	zcmr, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "results":
			var zajw uint32
			zajw, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Results) >= int(zajw) {
				z.Results = (z.Results)[:zajw]
			} else {
				z.Results = make([]string, zajw)
			}
			for zxvk := range z.Results {
				z.Results[zxvk], bts, err = msgp.ReadStringBytes(bts)
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
func (z *MetricExpandItem) Msgsize() (s int) {
	s = 1 + 8 + msgp.ArrayHeaderSize
	for zxvk := range z.Results {
		s += msgp.StringPrefixSize + len(z.Results[zxvk])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricFindItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zcua uint32
	zcua, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcua > 0 {
		zcua--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "text":
			z.Text, err = dc.ReadString()
			if err != nil {
				return
			}
		case "expandable":
			z.Expandable, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "leaf":
			z.Leaf, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "path":
			z.Path, err = dc.ReadString()
			if err != nil {
				return
			}
		case "allowChildren":
			z.AllowChildren, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "id":
			z.Id, err = dc.ReadString()
			if err != nil {
				return
			}
		case "uniqueid":
			z.UniqueId, err = dc.ReadString()
			if err != nil {
				return
			}
		case "tags":
			var zxhx uint32
			zxhx, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zxhx) {
				z.Tags = (z.Tags)[:zxhx]
			} else {
				z.Tags = make([]*repr.Tag, zxhx)
			}
			for zwht := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zwht] = nil
				} else {
					if z.Tags[zwht] == nil {
						z.Tags[zwht] = new(repr.Tag)
					}
					err = z.Tags[zwht].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "meta_tags":
			var zlqf uint32
			zlqf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zlqf) {
				z.MetaTags = (z.MetaTags)[:zlqf]
			} else {
				z.MetaTags = make([]*repr.Tag, zlqf)
			}
			for zhct := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zhct] = nil
				} else {
					if z.MetaTags[zhct] == nil {
						z.MetaTags[zhct] = new(repr.Tag)
					}
					err = z.MetaTags[zhct].DecodeMsg(dc)
					if err != nil {
						return
					}
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
func (z *MetricFindItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "text"
	err = en.Append(0x89, 0xa4, 0x74, 0x65, 0x78, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Text)
	if err != nil {
		return
	}
	// write "expandable"
	err = en.Append(0xaa, 0x65, 0x78, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x62, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Expandable)
	if err != nil {
		return
	}
	// write "leaf"
	err = en.Append(0xa4, 0x6c, 0x65, 0x61, 0x66)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Leaf)
	if err != nil {
		return
	}
	// write "path"
	err = en.Append(0xa4, 0x70, 0x61, 0x74, 0x68)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Path)
	if err != nil {
		return
	}
	// write "allowChildren"
	err = en.Append(0xad, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x43, 0x68, 0x69, 0x6c, 0x64, 0x72, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.AllowChildren)
	if err != nil {
		return
	}
	// write "id"
	err = en.Append(0xa2, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Id)
	if err != nil {
		return
	}
	// write "uniqueid"
	err = en.Append(0xa8, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.UniqueId)
	if err != nil {
		return
	}
	// write "tags"
	err = en.Append(0xa4, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zwht := range z.Tags {
		if z.Tags[zwht] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zwht].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zhct := range z.MetaTags {
		if z.MetaTags[zhct] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zhct].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricFindItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "text"
	o = append(o, 0x89, 0xa4, 0x74, 0x65, 0x78, 0x74)
	o = msgp.AppendString(o, z.Text)
	// string "expandable"
	o = append(o, 0xaa, 0x65, 0x78, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x62, 0x6c, 0x65)
	o = msgp.AppendUint32(o, z.Expandable)
	// string "leaf"
	o = append(o, 0xa4, 0x6c, 0x65, 0x61, 0x66)
	o = msgp.AppendUint32(o, z.Leaf)
	// string "path"
	o = append(o, 0xa4, 0x70, 0x61, 0x74, 0x68)
	o = msgp.AppendString(o, z.Path)
	// string "allowChildren"
	o = append(o, 0xad, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x43, 0x68, 0x69, 0x6c, 0x64, 0x72, 0x65, 0x6e)
	o = msgp.AppendUint32(o, z.AllowChildren)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendString(o, z.Id)
	// string "uniqueid"
	o = append(o, 0xa8, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x69, 0x64)
	o = msgp.AppendString(o, z.UniqueId)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zwht := range z.Tags {
		if z.Tags[zwht] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zwht].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zhct := range z.MetaTags {
		if z.MetaTags[zhct] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zhct].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricFindItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zdaf uint32
	zdaf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zdaf > 0 {
		zdaf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "text":
			z.Text, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "expandable":
			z.Expandable, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "leaf":
			z.Leaf, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "path":
			z.Path, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "allowChildren":
			z.AllowChildren, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "id":
			z.Id, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "uniqueid":
			z.UniqueId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zpks uint32
			zpks, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zpks) {
				z.Tags = (z.Tags)[:zpks]
			} else {
				z.Tags = make([]*repr.Tag, zpks)
			}
			for zwht := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zwht] = nil
				} else {
					if z.Tags[zwht] == nil {
						z.Tags[zwht] = new(repr.Tag)
					}
					bts, err = z.Tags[zwht].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "meta_tags":
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zjfb) {
				z.MetaTags = (z.MetaTags)[:zjfb]
			} else {
				z.MetaTags = make([]*repr.Tag, zjfb)
			}
			for zhct := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zhct] = nil
				} else {
					if z.MetaTags[zhct] == nil {
						z.MetaTags[zhct] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zhct].UnmarshalMsg(bts)
					if err != nil {
						return
					}
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
func (z *MetricFindItem) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Text) + 11 + msgp.Uint32Size + 5 + msgp.Uint32Size + 5 + msgp.StringPrefixSize + len(z.Path) + 14 + msgp.Uint32Size + 3 + msgp.StringPrefixSize + len(z.Id) + 9 + msgp.StringPrefixSize + len(z.UniqueId) + 5 + msgp.ArrayHeaderSize
	for zwht := range z.Tags {
		if z.Tags[zwht] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zwht].Msgsize()
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zhct := range z.MetaTags {
		if z.MetaTags[zhct] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zhct].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricFindItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zeff uint32
	zeff, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zeff > 0 {
		zeff--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zrsw uint32
			zrsw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zrsw) {
				z.Items = (z.Items)[:zrsw]
			} else {
				z.Items = make([]*MetricFindItem, zrsw)
			}
			for zcxo := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zcxo] = nil
				} else {
					if z.Items[zcxo] == nil {
						z.Items[zcxo] = new(MetricFindItem)
					}
					err = z.Items[zcxo].DecodeMsg(dc)
					if err != nil {
						return
					}
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
func (z *MetricFindItemList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "items"
	err = en.Append(0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Items)))
	if err != nil {
		return
	}
	for zcxo := range z.Items {
		if z.Items[zcxo] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zcxo].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricFindItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zcxo := range z.Items {
		if z.Items[zcxo] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zcxo].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricFindItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zxpk uint32
	zxpk, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zxpk > 0 {
		zxpk--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zdnj uint32
			zdnj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zdnj) {
				z.Items = (z.Items)[:zdnj]
			} else {
				z.Items = make([]*MetricFindItem, zdnj)
			}
			for zcxo := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zcxo] = nil
				} else {
					if z.Items[zcxo] == nil {
						z.Items[zcxo] = new(MetricFindItem)
					}
					bts, err = z.Items[zcxo].UnmarshalMsg(bts)
					if err != nil {
						return
					}
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
func (z *MetricFindItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zcxo := range z.Items {
		if z.Items[zcxo] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zcxo].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricTagItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zobc uint32
	zobc, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zobc > 0 {
		zobc--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "id":
			z.Id, err = dc.ReadString()
			if err != nil {
				return
			}
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "value":
			z.Value, err = dc.ReadString()
			if err != nil {
				return
			}
		case "is_meta":
			z.IsMeta, err = dc.ReadBool()
			if err != nil {
				return
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
func (z *MetricTagItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "id"
	err = en.Append(0x84, 0xa2, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Id)
	if err != nil {
		return
	}
	// write "name"
	err = en.Append(0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "value"
	err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Value)
	if err != nil {
		return
	}
	// write "is_meta"
	err = en.Append(0xa7, 0x69, 0x73, 0x5f, 0x6d, 0x65, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsMeta)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricTagItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "id"
	o = append(o, 0x84, 0xa2, 0x69, 0x64)
	o = msgp.AppendString(o, z.Id)
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "value"
	o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendString(o, z.Value)
	// string "is_meta"
	o = append(o, 0xa7, 0x69, 0x73, 0x5f, 0x6d, 0x65, 0x74, 0x61)
	o = msgp.AppendBool(o, z.IsMeta)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricTagItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zsnv uint32
	zsnv, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zsnv > 0 {
		zsnv--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "id":
			z.Id, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "value":
			z.Value, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "is_meta":
			z.IsMeta, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
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
func (z *MetricTagItem) Msgsize() (s int) {
	s = 1 + 3 + msgp.StringPrefixSize + len(z.Id) + 5 + msgp.StringPrefixSize + len(z.Name) + 6 + msgp.StringPrefixSize + len(z.Value) + 8 + msgp.BoolSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricTagItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zema uint32
	zema, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zema > 0 {
		zema--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zpez uint32
			zpez, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zpez) {
				z.Items = (z.Items)[:zpez]
			} else {
				z.Items = make([]*MetricTagItem, zpez)
			}
			for zkgt := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zkgt] = nil
				} else {
					if z.Items[zkgt] == nil {
						z.Items[zkgt] = new(MetricTagItem)
					}
					err = z.Items[zkgt].DecodeMsg(dc)
					if err != nil {
						return
					}
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
func (z *MetricTagItemList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "items"
	err = en.Append(0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Items)))
	if err != nil {
		return
	}
	for zkgt := range z.Items {
		if z.Items[zkgt] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zkgt].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricTagItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zkgt := range z.Items {
		if z.Items[zkgt] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zkgt].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricTagItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zqke uint32
	zqke, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zqke > 0 {
		zqke--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zqyh uint32
			zqyh, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zqyh) {
				z.Items = (z.Items)[:zqyh]
			} else {
				z.Items = make([]*MetricTagItem, zqyh)
			}
			for zkgt := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zkgt] = nil
				} else {
					if z.Items[zkgt] == nil {
						z.Items[zkgt] = new(MetricTagItem)
					}
					bts, err = z.Items[zkgt].UnmarshalMsg(bts)
					if err != nil {
						return
					}
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
func (z *MetricTagItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zkgt := range z.Items {
		if z.Items[zkgt] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zkgt].Msgsize()
		}
	}
	return
}

package repr

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *HashMode) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zxvk int32
		zxvk, err = dc.ReadInt32()
		(*z) = HashMode(zxvk)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z HashMode) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt32(int32(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z HashMode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt32(o, int32(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *HashMode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zbzg int32
		zbzg, bts, err = msgp.ReadInt32Bytes(bts)
		(*z) = HashMode(zbzg)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z HashMode) Msgsize() (s int) {
	s = msgp.Int32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatName) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zajw uint32
	zajw, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zajw > 0 {
		zajw--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Key":
			z.Key, err = dc.ReadString()
			if err != nil {
				return
			}
		case "XXX_uniqueId":
			z.XXX_uniqueId, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "XXX_uniqueIdstr":
			z.XXX_uniqueIdstr, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "TagMode":
			{
				var zwht int32
				zwht, err = dc.ReadInt32()
				z.TagMode = TagMode(zwht)
			}
			if err != nil {
				return
			}
		case "HashMode":
			{
				var zhct int32
				zhct, err = dc.ReadInt32()
				z.HashMode = HashMode(zhct)
			}
			if err != nil {
				return
			}
		case "Tags":
			var zcua uint32
			zcua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zcua) {
				z.Tags = (z.Tags)[:zcua]
			} else {
				z.Tags = make([]*Tag, zcua)
			}
			for zbai := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zbai] = nil
				} else {
					if z.Tags[zbai] == nil {
						z.Tags[zbai] = new(Tag)
					}
					var zxhx uint32
					zxhx, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zxhx > 0 {
						zxhx--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Name":
							z.Tags[zbai].Name, err = dc.ReadString()
							if err != nil {
								return
							}
						case "Value":
							z.Tags[zbai].Value, err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zlqf uint32
			zlqf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zlqf) {
				z.MetaTags = (z.MetaTags)[:zlqf]
			} else {
				z.MetaTags = make([]*Tag, zlqf)
			}
			for zcmr := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zcmr] = nil
				} else {
					if z.MetaTags[zcmr] == nil {
						z.MetaTags[zcmr] = new(Tag)
					}
					var zdaf uint32
					zdaf, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zdaf > 0 {
						zdaf--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Name":
							z.MetaTags[zcmr].Name, err = dc.ReadString()
							if err != nil {
								return
							}
						case "Value":
							z.MetaTags[zcmr].Value, err = dc.ReadString()
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
func (z *StatName) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "Key"
	err = en.Append(0x89, 0xa3, 0x4b, 0x65, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Key)
	if err != nil {
		return
	}
	// write "XXX_uniqueId"
	err = en.Append(0xac, 0x58, 0x58, 0x58, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.XXX_uniqueId)
	if err != nil {
		return
	}
	// write "XXX_uniqueIdstr"
	err = en.Append(0xaf, 0x58, 0x58, 0x58, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x49, 0x64, 0x73, 0x74, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteString(z.XXX_uniqueIdstr)
	if err != nil {
		return
	}
	// write "Resolution"
	err = en.Append(0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Resolution)
	if err != nil {
		return
	}
	// write "Ttl"
	err = en.Append(0xa3, 0x54, 0x74, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Ttl)
	if err != nil {
		return
	}
	// write "TagMode"
	err = en.Append(0xa7, 0x54, 0x61, 0x67, 0x4d, 0x6f, 0x64, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt32(int32(z.TagMode))
	if err != nil {
		return
	}
	// write "HashMode"
	err = en.Append(0xa8, 0x48, 0x61, 0x73, 0x68, 0x4d, 0x6f, 0x64, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt32(int32(z.HashMode))
	if err != nil {
		return
	}
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "Name"
			err = en.Append(0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteString(z.Tags[zbai].Name)
			if err != nil {
				return
			}
			// write "Value"
			err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteString(z.Tags[zbai].Value)
			if err != nil {
				return
			}
		}
	}
	// write "MetaTags"
	err = en.Append(0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "Name"
			err = en.Append(0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteString(z.MetaTags[zcmr].Name)
			if err != nil {
				return
			}
			// write "Value"
			err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteString(z.MetaTags[zcmr].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *StatName) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "Key"
	o = append(o, 0x89, 0xa3, 0x4b, 0x65, 0x79)
	o = msgp.AppendString(o, z.Key)
	// string "XXX_uniqueId"
	o = append(o, 0xac, 0x58, 0x58, 0x58, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.XXX_uniqueId)
	// string "XXX_uniqueIdstr"
	o = append(o, 0xaf, 0x58, 0x58, 0x58, 0x5f, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x49, 0x64, 0x73, 0x74, 0x72)
	o = msgp.AppendString(o, z.XXX_uniqueIdstr)
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "TagMode"
	o = append(o, 0xa7, 0x54, 0x61, 0x67, 0x4d, 0x6f, 0x64, 0x65)
	o = msgp.AppendInt32(o, int32(z.TagMode))
	// string "HashMode"
	o = append(o, 0xa8, 0x48, 0x61, 0x73, 0x68, 0x4d, 0x6f, 0x64, 0x65)
	o = msgp.AppendInt32(o, int32(z.HashMode))
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "Name"
			o = append(o, 0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
			o = msgp.AppendString(o, z.Tags[zbai].Name)
			// string "Value"
			o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendString(o, z.Tags[zbai].Value)
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "Name"
			o = append(o, 0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
			o = msgp.AppendString(o, z.MetaTags[zcmr].Name)
			// string "Value"
			o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendString(o, z.MetaTags[zcmr].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatName) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zpks uint32
	zpks, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zpks > 0 {
		zpks--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Key":
			z.Key, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "XXX_uniqueId":
			z.XXX_uniqueId, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "XXX_uniqueIdstr":
			z.XXX_uniqueIdstr, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "TagMode":
			{
				var zjfb int32
				zjfb, bts, err = msgp.ReadInt32Bytes(bts)
				z.TagMode = TagMode(zjfb)
			}
			if err != nil {
				return
			}
		case "HashMode":
			{
				var zcxo int32
				zcxo, bts, err = msgp.ReadInt32Bytes(bts)
				z.HashMode = HashMode(zcxo)
			}
			if err != nil {
				return
			}
		case "Tags":
			var zeff uint32
			zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zeff) {
				z.Tags = (z.Tags)[:zeff]
			} else {
				z.Tags = make([]*Tag, zeff)
			}
			for zbai := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zbai] = nil
				} else {
					if z.Tags[zbai] == nil {
						z.Tags[zbai] = new(Tag)
					}
					var zrsw uint32
					zrsw, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zrsw > 0 {
						zrsw--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Name":
							z.Tags[zbai].Name, bts, err = msgp.ReadStringBytes(bts)
							if err != nil {
								return
							}
						case "Value":
							z.Tags[zbai].Value, bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zxpk uint32
			zxpk, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zxpk) {
				z.MetaTags = (z.MetaTags)[:zxpk]
			} else {
				z.MetaTags = make([]*Tag, zxpk)
			}
			for zcmr := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zcmr] = nil
				} else {
					if z.MetaTags[zcmr] == nil {
						z.MetaTags[zcmr] = new(Tag)
					}
					var zdnj uint32
					zdnj, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zdnj > 0 {
						zdnj--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Name":
							z.MetaTags[zcmr].Name, bts, err = msgp.ReadStringBytes(bts)
							if err != nil {
								return
							}
						case "Value":
							z.MetaTags[zcmr].Value, bts, err = msgp.ReadStringBytes(bts)
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
func (z *StatName) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Key) + 13 + msgp.Uint64Size + 16 + msgp.StringPrefixSize + len(z.XXX_uniqueIdstr) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 8 + msgp.Int32Size + 9 + msgp.Int32Size + 5 + msgp.ArrayHeaderSize
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.StringPrefixSize + len(z.Tags[zbai].Name) + 6 + msgp.StringPrefixSize + len(z.Tags[zbai].Value)
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.StringPrefixSize + len(z.MetaTags[zcmr].Name) + 6 + msgp.StringPrefixSize + len(z.MetaTags[zcmr].Value)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatRepr) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Name":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Name = nil
			} else {
				if z.Name == nil {
					z.Name = new(StatName)
				}
				err = z.Name.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Min":
			z.Min, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Max":
			z.Max, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Last":
			z.Last, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Count":
			z.Count, err = dc.ReadInt64()
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
func (z *StatRepr) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "Name"
	err = en.Append(0x87, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	if z.Name == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Name.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Time"
	err = en.Append(0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Min"
	err = en.Append(0xa3, 0x4d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Min)
	if err != nil {
		return
	}
	// write "Max"
	err = en.Append(0xa3, 0x4d, 0x61, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Max)
	if err != nil {
		return
	}
	// write "Last"
	err = en.Append(0xa4, 0x4c, 0x61, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Last)
	if err != nil {
		return
	}
	// write "Sum"
	err = en.Append(0xa3, 0x53, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
	if err != nil {
		return
	}
	// write "Count"
	err = en.Append(0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Count)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *StatRepr) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "Name"
	o = append(o, 0x87, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
	if z.Name == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Name.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Time"
	o = append(o, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Min"
	o = append(o, 0xa3, 0x4d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "Max"
	o = append(o, 0xa3, 0x4d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "Last"
	o = append(o, 0xa4, 0x4c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "Sum"
	o = append(o, 0xa3, 0x53, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "Count"
	o = append(o, 0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatRepr) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Name":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Name = nil
			} else {
				if z.Name == nil {
					z.Name = new(StatName)
				}
				bts, err = z.Name.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Min":
			z.Min, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Max":
			z.Max, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Last":
			z.Last, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Count":
			z.Count, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *StatRepr) Msgsize() (s int) {
	s = 1 + 5
	if z.Name == nil {
		s += msgp.NilSize
	} else {
		s += z.Name.Msgsize()
	}
	s += 5 + msgp.Int64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Tag) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zkgt uint32
	zkgt, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zkgt > 0 {
		zkgt--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Value":
			z.Value, err = dc.ReadString()
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
func (z Tag) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Name"
	err = en.Append(0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "Value"
	err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Value)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Tag) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Name"
	o = append(o, 0x82, 0xa4, 0x4e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendString(o, z.Value)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Tag) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zema uint32
	zema, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zema > 0 {
		zema--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadStringBytes(bts)
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
func (z Tag) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Name) + 6 + msgp.StringPrefixSize + len(z.Value)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TagMode) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zpez int32
		zpez, err = dc.ReadInt32()
		(*z) = TagMode(zpez)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z TagMode) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteInt32(int32(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z TagMode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt32(o, int32(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TagMode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zqke int32
		zqke, bts, err = msgp.ReadInt32Bytes(bts)
		(*z) = TagMode(zqke)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z TagMode) Msgsize() (s int) {
	s = msgp.Int32Size
	return
}

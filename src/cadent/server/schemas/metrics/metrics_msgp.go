package metrics

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/schemas/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *DataPoint) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxvk uint32
	zxvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxvk > 0 {
		zxvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "time":
			z.Time, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "value":
			z.Value, err = dc.ReadFloat64()
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
func (z DataPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "time"
	err = en.Append(0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Time)
	if err != nil {
		return
	}
	// write "value"
	err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Value)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z DataPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "time"
	o = append(o, 0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendUint32(o, z.Time)
	// string "value"
	o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendFloat64(o, z.Value)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DataPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zbzg uint32
	zbzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zbzg > 0 {
		zbzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "time":
			z.Time, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "value":
			z.Value, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z DataPoint) Msgsize() (s int) {
	s = 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *DataPoints) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "points":
			var zajw uint32
			zajw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Points) >= int(zajw) {
				z.Points = (z.Points)[:zajw]
			} else {
				z.Points = make([]*DataPoint, zajw)
			}
			for zbai := range z.Points {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Points[zbai] = nil
				} else {
					if z.Points[zbai] == nil {
						z.Points[zbai] = new(DataPoint)
					}
					var zwht uint32
					zwht, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zwht > 0 {
						zwht--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.Points[zbai].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "value":
							z.Points[zbai].Value, err = dc.ReadFloat64()
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
func (z *DataPoints) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "points"
	err = en.Append(0x81, 0xa6, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Points)))
	if err != nil {
		return
	}
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "time"
			err = en.Append(0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Points[zbai].Time)
			if err != nil {
				return
			}
			// write "value"
			err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.Points[zbai].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DataPoints) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "points"
	o = append(o, 0x81, 0xa6, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Points)))
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "time"
			o = append(o, 0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.Points[zbai].Time)
			// string "value"
			o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.Points[zbai].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DataPoints) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zhct uint32
	zhct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "points":
			var zcua uint32
			zcua, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Points) >= int(zcua) {
				z.Points = (z.Points)[:zcua]
			} else {
				z.Points = make([]*DataPoint, zcua)
			}
			for zbai := range z.Points {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Points[zbai] = nil
				} else {
					if z.Points[zbai] == nil {
						z.Points[zbai] = new(DataPoint)
					}
					var zxhx uint32
					zxhx, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zxhx > 0 {
						zxhx--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.Points[zbai].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "value":
							z.Points[zbai].Value, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *DataPoints) Msgsize() (s int) {
	s = 1 + 7 + msgp.ArrayHeaderSize
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *GraphiteApiItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
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
		case "target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "in_cache":
			z.InCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "datapoints":
			var zpks uint32
			zpks, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Datapoints) >= int(zpks) {
				z.Datapoints = (z.Datapoints)[:zpks]
			} else {
				z.Datapoints = make([]*DataPoint, zpks)
			}
			for zlqf := range z.Datapoints {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Datapoints[zlqf] = nil
				} else {
					if z.Datapoints[zlqf] == nil {
						z.Datapoints[zlqf] = new(DataPoint)
					}
					var zjfb uint32
					zjfb, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zjfb > 0 {
						zjfb--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.Datapoints[zlqf].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "value":
							z.Datapoints[zlqf].Value, err = dc.ReadFloat64()
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
func (z *GraphiteApiItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "target"
	err = en.Append(0x84, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "in_cache"
	err = en.Append(0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.InCache)
	if err != nil {
		return
	}
	// write "using_cache"
	err = en.Append(0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.UsingCache)
	if err != nil {
		return
	}
	// write "datapoints"
	err = en.Append(0xaa, 0x64, 0x61, 0x74, 0x61, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Datapoints)))
	if err != nil {
		return
	}
	for zlqf := range z.Datapoints {
		if z.Datapoints[zlqf] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "time"
			err = en.Append(0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Datapoints[zlqf].Time)
			if err != nil {
				return
			}
			// write "value"
			err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.Datapoints[zlqf].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *GraphiteApiItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "target"
	o = append(o, 0x84, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "in_cache"
	o = append(o, 0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.InCache)
	// string "using_cache"
	o = append(o, 0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.UsingCache)
	// string "datapoints"
	o = append(o, 0xaa, 0x64, 0x61, 0x74, 0x61, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Datapoints)))
	for zlqf := range z.Datapoints {
		if z.Datapoints[zlqf] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "time"
			o = append(o, 0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.Datapoints[zlqf].Time)
			// string "value"
			o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.Datapoints[zlqf].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *GraphiteApiItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcxo uint32
	zcxo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcxo > 0 {
		zcxo--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "in_cache":
			z.InCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "datapoints":
			var zeff uint32
			zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Datapoints) >= int(zeff) {
				z.Datapoints = (z.Datapoints)[:zeff]
			} else {
				z.Datapoints = make([]*DataPoint, zeff)
			}
			for zlqf := range z.Datapoints {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Datapoints[zlqf] = nil
				} else {
					if z.Datapoints[zlqf] == nil {
						z.Datapoints[zlqf] = new(DataPoint)
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
						case "time":
							z.Datapoints[zlqf].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "value":
							z.Datapoints[zlqf].Value, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *GraphiteApiItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 9 + msgp.BoolSize + 12 + msgp.BoolSize + 11 + msgp.ArrayHeaderSize
	for zlqf := range z.Datapoints {
		if z.Datapoints[zlqf] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *GraphiteApiItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zdnj uint32
	zdnj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zdnj > 0 {
		zdnj--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zobc uint32
			zobc, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zobc) {
				z.Items = (z.Items)[:zobc]
			} else {
				z.Items = make([]*GraphiteApiItem, zobc)
			}
			for zxpk := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zxpk] = nil
				} else {
					if z.Items[zxpk] == nil {
						z.Items[zxpk] = new(GraphiteApiItem)
					}
					err = z.Items[zxpk].DecodeMsg(dc)
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
func (z *GraphiteApiItemList) EncodeMsg(en *msgp.Writer) (err error) {
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
	for zxpk := range z.Items {
		if z.Items[zxpk] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zxpk].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *GraphiteApiItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zxpk := range z.Items {
		if z.Items[zxpk] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zxpk].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *GraphiteApiItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "items":
			var zkgt uint32
			zkgt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zkgt) {
				z.Items = (z.Items)[:zkgt]
			} else {
				z.Items = make([]*GraphiteApiItem, zkgt)
			}
			for zxpk := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zxpk] = nil
				} else {
					if z.Items[zxpk] == nil {
						z.Items[zxpk] = new(GraphiteApiItem)
					}
					bts, err = z.Items[zxpk].UnmarshalMsg(bts)
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
func (z *GraphiteApiItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zxpk := range z.Items {
		if z.Items[zxpk] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zxpk].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawDataPoint) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "time":
			z.Time, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "min":
			z.Min, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "max":
			z.Max, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "last":
			z.Last, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "count":
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
func (z *RawDataPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "time"
	err = en.Append(0x86, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Time)
	if err != nil {
		return
	}
	// write "sum"
	err = en.Append(0xa3, 0x73, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
	if err != nil {
		return
	}
	// write "min"
	err = en.Append(0xa3, 0x6d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Min)
	if err != nil {
		return
	}
	// write "max"
	err = en.Append(0xa3, 0x6d, 0x61, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Max)
	if err != nil {
		return
	}
	// write "last"
	err = en.Append(0xa4, 0x6c, 0x61, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Last)
	if err != nil {
		return
	}
	// write "count"
	err = en.Append(0xa5, 0x63, 0x6f, 0x75, 0x6e, 0x74)
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
func (z *RawDataPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "time"
	o = append(o, 0x86, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendUint32(o, z.Time)
	// string "sum"
	o = append(o, 0xa3, 0x73, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "min"
	o = append(o, 0xa3, 0x6d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "max"
	o = append(o, 0xa3, 0x6d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "last"
	o = append(o, 0xa4, 0x6c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "count"
	o = append(o, 0xa5, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawDataPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zpez uint32
	zpez, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zpez > 0 {
		zpez--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "time":
			z.Time, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "min":
			z.Min, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "max":
			z.Max, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "last":
			z.Last, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "count":
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
func (z *RawDataPoint) Msgsize() (s int) {
	s = 1 + 5 + msgp.Uint32Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 6 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawRenderItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zywj uint32
	zywj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zywj > 0 {
		zywj--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "id":
			z.Id, err = dc.ReadString()
			if err != nil {
				return
			}
		case "real_start":
			z.RealStart, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "real_end":
			z.RealEnd, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "start":
			z.Start, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "end":
			z.End, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "step":
			z.Step, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "agg_func":
			z.AggFunc, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "tags":
			var zjpj uint32
			zjpj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zjpj) {
				z.Tags = (z.Tags)[:zjpj]
			} else {
				z.Tags = make([]*repr.Tag, zjpj)
			}
			for zqke := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zqke] = nil
				} else {
					if z.Tags[zqke] == nil {
						z.Tags[zqke] = new(repr.Tag)
					}
					err = z.Tags[zqke].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "meta_tags":
			var zzpf uint32
			zzpf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zzpf) {
				z.MetaTags = (z.MetaTags)[:zzpf]
			} else {
				z.MetaTags = make([]*repr.Tag, zzpf)
			}
			for zqyh := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zqyh] = nil
				} else {
					if z.MetaTags[zqyh] == nil {
						z.MetaTags[zqyh] = new(repr.Tag)
					}
					err = z.MetaTags[zqyh].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "data":
			var zrfe uint32
			zrfe, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zrfe) {
				z.Data = (z.Data)[:zrfe]
			} else {
				z.Data = make([]*RawDataPoint, zrfe)
			}
			for zyzr := range z.Data {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Data[zyzr] = nil
				} else {
					if z.Data[zyzr] == nil {
						z.Data[zyzr] = new(RawDataPoint)
					}
					err = z.Data[zyzr].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "in_cache":
			z.InCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, err = dc.ReadBool()
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
func (z *RawRenderItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 13
	// write "metric"
	err = en.Append(0x8d, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "real_start"
	err = en.Append(0xaa, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealStart)
	if err != nil {
		return
	}
	// write "real_end"
	err = en.Append(0xa8, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x65, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealEnd)
	if err != nil {
		return
	}
	// write "start"
	err = en.Append(0xa5, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Start)
	if err != nil {
		return
	}
	// write "end"
	err = en.Append(0xa3, 0x65, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.End)
	if err != nil {
		return
	}
	// write "step"
	err = en.Append(0xa4, 0x73, 0x74, 0x65, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Step)
	if err != nil {
		return
	}
	// write "agg_func"
	err = en.Append(0xa8, 0x61, 0x67, 0x67, 0x5f, 0x66, 0x75, 0x6e, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.AggFunc)
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
	for zqke := range z.Tags {
		if z.Tags[zqke] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zqke].EncodeMsg(en)
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
	for zqyh := range z.MetaTags {
		if z.MetaTags[zqyh] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zqyh].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "data"
	err = en.Append(0xa4, 0x64, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Data)))
	if err != nil {
		return
	}
	for zyzr := range z.Data {
		if z.Data[zyzr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Data[zyzr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "in_cache"
	err = en.Append(0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.InCache)
	if err != nil {
		return
	}
	// write "using_cache"
	err = en.Append(0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.UsingCache)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawRenderItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 13
	// string "metric"
	o = append(o, 0x8d, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendString(o, z.Id)
	// string "real_start"
	o = append(o, 0xaa, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.RealStart)
	// string "real_end"
	o = append(o, 0xa8, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x65, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.RealEnd)
	// string "start"
	o = append(o, 0xa5, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.Start)
	// string "end"
	o = append(o, 0xa3, 0x65, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.End)
	// string "step"
	o = append(o, 0xa4, 0x73, 0x74, 0x65, 0x70)
	o = msgp.AppendUint32(o, z.Step)
	// string "agg_func"
	o = append(o, 0xa8, 0x61, 0x67, 0x67, 0x5f, 0x66, 0x75, 0x6e, 0x63)
	o = msgp.AppendUint32(o, z.AggFunc)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zqke := range z.Tags {
		if z.Tags[zqke] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zqke].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zqyh := range z.MetaTags {
		if z.MetaTags[zqyh] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zqyh].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "data"
	o = append(o, 0xa4, 0x64, 0x61, 0x74, 0x61)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Data)))
	for zyzr := range z.Data {
		if z.Data[zyzr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Data[zyzr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "in_cache"
	o = append(o, 0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.InCache)
	// string "using_cache"
	o = append(o, 0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.UsingCache)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawRenderItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zgmo uint32
	zgmo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zgmo > 0 {
		zgmo--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "id":
			z.Id, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "real_start":
			z.RealStart, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "real_end":
			z.RealEnd, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "start":
			z.Start, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "end":
			z.End, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "step":
			z.Step, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "agg_func":
			z.AggFunc, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var ztaf uint32
			ztaf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(ztaf) {
				z.Tags = (z.Tags)[:ztaf]
			} else {
				z.Tags = make([]*repr.Tag, ztaf)
			}
			for zqke := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zqke] = nil
				} else {
					if z.Tags[zqke] == nil {
						z.Tags[zqke] = new(repr.Tag)
					}
					bts, err = z.Tags[zqke].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "meta_tags":
			var zeth uint32
			zeth, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zeth) {
				z.MetaTags = (z.MetaTags)[:zeth]
			} else {
				z.MetaTags = make([]*repr.Tag, zeth)
			}
			for zqyh := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zqyh] = nil
				} else {
					if z.MetaTags[zqyh] == nil {
						z.MetaTags[zqyh] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zqyh].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "data":
			var zsbz uint32
			zsbz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zsbz) {
				z.Data = (z.Data)[:zsbz]
			} else {
				z.Data = make([]*RawDataPoint, zsbz)
			}
			for zyzr := range z.Data {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Data[zyzr] = nil
				} else {
					if z.Data[zyzr] == nil {
						z.Data[zyzr] = new(RawDataPoint)
					}
					bts, err = z.Data[zyzr].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "in_cache":
			z.InCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, bts, err = msgp.ReadBoolBytes(bts)
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
func (z *RawRenderItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Metric) + 3 + msgp.StringPrefixSize + len(z.Id) + 11 + msgp.Uint32Size + 9 + msgp.Uint32Size + 6 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.Uint32Size + 9 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zqke := range z.Tags {
		if z.Tags[zqke] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zqke].Msgsize()
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zqyh := range z.MetaTags {
		if z.MetaTags[zqyh] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zqyh].Msgsize()
		}
	}
	s += 5 + msgp.ArrayHeaderSize
	for zyzr := range z.Data {
		if z.Data[zyzr] == nil {
			s += msgp.NilSize
		} else {
			s += z.Data[zyzr].Msgsize()
		}
	}
	s += 9 + msgp.BoolSize + 12 + msgp.BoolSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawRenderItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zawn uint32
	zawn, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zawn > 0 {
		zawn--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zwel uint32
			zwel, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zwel) {
				z.Items = (z.Items)[:zwel]
			} else {
				z.Items = make([]*RawRenderItem, zwel)
			}
			for zrjx := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zrjx] = nil
				} else {
					if z.Items[zrjx] == nil {
						z.Items[zrjx] = new(RawRenderItem)
					}
					err = z.Items[zrjx].DecodeMsg(dc)
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
func (z *RawRenderItemList) EncodeMsg(en *msgp.Writer) (err error) {
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
	for zrjx := range z.Items {
		if z.Items[zrjx] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zrjx].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawRenderItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zrjx := range z.Items {
		if z.Items[zrjx] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zrjx].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawRenderItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zrbe uint32
	zrbe, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zrbe > 0 {
		zrbe--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zmfd uint32
			zmfd, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zmfd) {
				z.Items = (z.Items)[:zmfd]
			} else {
				z.Items = make([]*RawRenderItem, zmfd)
			}
			for zrjx := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zrjx] = nil
				} else {
					if z.Items[zrjx] == nil {
						z.Items[zrjx] = new(RawRenderItem)
					}
					bts, err = z.Items[zrjx].UnmarshalMsg(bts)
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
func (z *RawRenderItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zrjx := range z.Items {
		if z.Items[zrjx] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zrjx].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RenderItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zelx uint32
	zelx, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zelx > 0 {
		zelx--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "dataPoints":
			var zbal uint32
			zbal, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.DataPoints) >= int(zbal) {
				z.DataPoints = (z.DataPoints)[:zbal]
			} else {
				z.DataPoints = make([]*DataPoint, zbal)
			}
			for zzdc := range z.DataPoints {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.DataPoints[zzdc] = nil
				} else {
					if z.DataPoints[zzdc] == nil {
						z.DataPoints[zzdc] = new(DataPoint)
					}
					var zjqz uint32
					zjqz, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zjqz > 0 {
						zjqz--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.DataPoints[zzdc].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "value":
							z.DataPoints[zzdc].Value, err = dc.ReadFloat64()
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
func (z *RenderItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "target"
	err = en.Append(0x82, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "dataPoints"
	err = en.Append(0xaa, 0x64, 0x61, 0x74, 0x61, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.DataPoints)))
	if err != nil {
		return
	}
	for zzdc := range z.DataPoints {
		if z.DataPoints[zzdc] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "time"
			err = en.Append(0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.DataPoints[zzdc].Time)
			if err != nil {
				return
			}
			// write "value"
			err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.DataPoints[zzdc].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RenderItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "target"
	o = append(o, 0x82, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "dataPoints"
	o = append(o, 0xaa, 0x64, 0x61, 0x74, 0x61, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.DataPoints)))
	for zzdc := range z.DataPoints {
		if z.DataPoints[zzdc] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "time"
			o = append(o, 0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.DataPoints[zzdc].Time)
			// string "value"
			o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.DataPoints[zzdc].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RenderItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zkct uint32
	zkct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zkct > 0 {
		zkct--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "dataPoints":
			var ztmt uint32
			ztmt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.DataPoints) >= int(ztmt) {
				z.DataPoints = (z.DataPoints)[:ztmt]
			} else {
				z.DataPoints = make([]*DataPoint, ztmt)
			}
			for zzdc := range z.DataPoints {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.DataPoints[zzdc] = nil
				} else {
					if z.DataPoints[zzdc] == nil {
						z.DataPoints[zzdc] = new(DataPoint)
					}
					var ztco uint32
					ztco, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for ztco > 0 {
						ztco--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.DataPoints[zzdc].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "value":
							z.DataPoints[zzdc].Value, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *RenderItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 11 + msgp.ArrayHeaderSize
	for zzdc := range z.DataPoints {
		if z.DataPoints[zzdc] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RenderItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ztyy uint32
	ztyy, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ztyy > 0 {
		ztyy--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zinl uint32
			zinl, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zinl) {
				z.Items = (z.Items)[:zinl]
			} else {
				z.Items = make([]*RenderItem, zinl)
			}
			for zana := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zana] = nil
				} else {
					if z.Items[zana] == nil {
						z.Items[zana] = new(RenderItem)
					}
					err = z.Items[zana].DecodeMsg(dc)
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
func (z *RenderItemList) EncodeMsg(en *msgp.Writer) (err error) {
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
	for zana := range z.Items {
		if z.Items[zana] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zana].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RenderItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zana := range z.Items {
		if z.Items[zana] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zana].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RenderItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zare uint32
	zare, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zare > 0 {
		zare--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zljy uint32
			zljy, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zljy) {
				z.Items = (z.Items)[:zljy]
			} else {
				z.Items = make([]*RenderItem, zljy)
			}
			for zana := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zana] = nil
				} else {
					if z.Items[zana] == nil {
						z.Items[zana] = new(RenderItem)
					}
					bts, err = z.Items[zana].UnmarshalMsg(bts)
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
func (z *RenderItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zana := range z.Items {
		if z.Items[zana] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zana].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WhisperDataItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zrsc uint32
	zrsc, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zrsc > 0 {
		zrsc--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "in_cache":
			z.InCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "data":
			var zctn uint32
			zctn, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zctn) {
				z.Data = (z.Data)[:zctn]
			} else {
				z.Data = make([]*DataPoint, zctn)
			}
			for zixj := range z.Data {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Data[zixj] = nil
				} else {
					if z.Data[zixj] == nil {
						z.Data[zixj] = new(DataPoint)
					}
					var zswy uint32
					zswy, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zswy > 0 {
						zswy--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.Data[zixj].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "value":
							z.Data[zixj].Value, err = dc.ReadFloat64()
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
func (z *WhisperDataItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "target"
	err = en.Append(0x85, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "in_cache"
	err = en.Append(0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.InCache)
	if err != nil {
		return
	}
	// write "using_cache"
	err = en.Append(0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.UsingCache)
	if err != nil {
		return
	}
	// write "uid"
	err = en.Append(0xa3, 0x75, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "data"
	err = en.Append(0xa4, 0x64, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Data)))
	if err != nil {
		return
	}
	for zixj := range z.Data {
		if z.Data[zixj] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "time"
			err = en.Append(0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Data[zixj].Time)
			if err != nil {
				return
			}
			// write "value"
			err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.Data[zixj].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *WhisperDataItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "target"
	o = append(o, 0x85, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "in_cache"
	o = append(o, 0xa8, 0x69, 0x6e, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.InCache)
	// string "using_cache"
	o = append(o, 0xab, 0x75, 0x73, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.UsingCache)
	// string "uid"
	o = append(o, 0xa3, 0x75, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "data"
	o = append(o, 0xa4, 0x64, 0x61, 0x74, 0x61)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Data)))
	for zixj := range z.Data {
		if z.Data[zixj] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "time"
			o = append(o, 0x82, 0xa4, 0x74, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.Data[zixj].Time)
			// string "value"
			o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.Data[zixj].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *WhisperDataItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var znsg uint32
	znsg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for znsg > 0 {
		znsg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "in_cache":
			z.InCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "using_cache":
			z.UsingCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "data":
			var zrus uint32
			zrus, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zrus) {
				z.Data = (z.Data)[:zrus]
			} else {
				z.Data = make([]*DataPoint, zrus)
			}
			for zixj := range z.Data {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Data[zixj] = nil
				} else {
					if z.Data[zixj] == nil {
						z.Data[zixj] = new(DataPoint)
					}
					var zsvm uint32
					zsvm, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zsvm > 0 {
						zsvm--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "time":
							z.Data[zixj].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "value":
							z.Data[zixj].Value, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *WhisperDataItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 9 + msgp.BoolSize + 12 + msgp.BoolSize + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.ArrayHeaderSize
	for zixj := range z.Data {
		if z.Data[zixj] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WhisperDataItemList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zfzb uint32
	zfzb, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zfzb > 0 {
		zfzb--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zsbo uint32
			zsbo, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zsbo) {
				z.Items = (z.Items)[:zsbo]
			} else {
				z.Items = make([]*WhisperDataItem, zsbo)
			}
			for zaoz := range z.Items {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Items[zaoz] = nil
				} else {
					if z.Items[zaoz] == nil {
						z.Items[zaoz] = new(WhisperDataItem)
					}
					err = z.Items[zaoz].DecodeMsg(dc)
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
func (z *WhisperDataItemList) EncodeMsg(en *msgp.Writer) (err error) {
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
	for zaoz := range z.Items {
		if z.Items[zaoz] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Items[zaoz].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *WhisperDataItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "items"
	o = append(o, 0x81, 0xa5, 0x69, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for zaoz := range z.Items {
		if z.Items[zaoz] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Items[zaoz].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *WhisperDataItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zjif uint32
	zjif, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zjif > 0 {
		zjif--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "items":
			var zqgz uint32
			zqgz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Items) >= int(zqgz) {
				z.Items = (z.Items)[:zqgz]
			} else {
				z.Items = make([]*WhisperDataItem, zqgz)
			}
			for zaoz := range z.Items {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Items[zaoz] = nil
				} else {
					if z.Items[zaoz] == nil {
						z.Items[zaoz] = new(WhisperDataItem)
					}
					bts, err = z.Items[zaoz].UnmarshalMsg(bts)
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
func (z *WhisperDataItemList) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zaoz := range z.Items {
		if z.Items[zaoz] == nil {
			s += msgp.NilSize
		} else {
			s += z.Items[zaoz].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WhisperRenderItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zmvo uint32
	zmvo, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zmvo > 0 {
		zmvo--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "real_start":
			z.RealStart, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "real_end":
			z.RealEnd, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "start":
			z.Start, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "end":
			z.End, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "from":
			z.From, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "to":
			z.To, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "step":
			z.Step, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "series":
			var zigk uint32
			zigk, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Series == nil && zigk > 0 {
				z.Series = make(map[string]*WhisperDataItem, zigk)
			} else if len(z.Series) > 0 {
				for key := range z.Series {
					delete(z.Series, key)
				}
			}
			for zigk > 0 {
				zigk--
				var zsnw string
				var ztls *WhisperDataItem
				zsnw, err = dc.ReadString()
				if err != nil {
					return
				}
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					ztls = nil
				} else {
					if ztls == nil {
						ztls = new(WhisperDataItem)
					}
					err = ztls.DecodeMsg(dc)
					if err != nil {
						return
					}
				}
				z.Series[zsnw] = ztls
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
func (z *WhisperRenderItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "real_start"
	err = en.Append(0x88, 0xaa, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealStart)
	if err != nil {
		return
	}
	// write "real_end"
	err = en.Append(0xa8, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x65, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealEnd)
	if err != nil {
		return
	}
	// write "start"
	err = en.Append(0xa5, 0x73, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Start)
	if err != nil {
		return
	}
	// write "end"
	err = en.Append(0xa3, 0x65, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.End)
	if err != nil {
		return
	}
	// write "from"
	err = en.Append(0xa4, 0x66, 0x72, 0x6f, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.From)
	if err != nil {
		return
	}
	// write "to"
	err = en.Append(0xa2, 0x74, 0x6f)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.To)
	if err != nil {
		return
	}
	// write "step"
	err = en.Append(0xa4, 0x73, 0x74, 0x65, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Step)
	if err != nil {
		return
	}
	// write "series"
	err = en.Append(0xa6, 0x73, 0x65, 0x72, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Series)))
	if err != nil {
		return
	}
	for zsnw, ztls := range z.Series {
		err = en.WriteString(zsnw)
		if err != nil {
			return
		}
		if ztls == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = ztls.EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *WhisperRenderItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "real_start"
	o = append(o, 0x88, 0xaa, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.RealStart)
	// string "real_end"
	o = append(o, 0xa8, 0x72, 0x65, 0x61, 0x6c, 0x5f, 0x65, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.RealEnd)
	// string "start"
	o = append(o, 0xa5, 0x73, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.Start)
	// string "end"
	o = append(o, 0xa3, 0x65, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.End)
	// string "from"
	o = append(o, 0xa4, 0x66, 0x72, 0x6f, 0x6d)
	o = msgp.AppendUint32(o, z.From)
	// string "to"
	o = append(o, 0xa2, 0x74, 0x6f)
	o = msgp.AppendUint32(o, z.To)
	// string "step"
	o = append(o, 0xa4, 0x73, 0x74, 0x65, 0x70)
	o = msgp.AppendUint32(o, z.Step)
	// string "series"
	o = append(o, 0xa6, 0x73, 0x65, 0x72, 0x69, 0x65, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Series)))
	for zsnw, ztls := range z.Series {
		o = msgp.AppendString(o, zsnw)
		if ztls == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = ztls.MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *WhisperRenderItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zopb uint32
	zopb, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zopb > 0 {
		zopb--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "real_start":
			z.RealStart, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "real_end":
			z.RealEnd, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "start":
			z.Start, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "end":
			z.End, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "from":
			z.From, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "to":
			z.To, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "step":
			z.Step, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "series":
			var zuop uint32
			zuop, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Series == nil && zuop > 0 {
				z.Series = make(map[string]*WhisperDataItem, zuop)
			} else if len(z.Series) > 0 {
				for key := range z.Series {
					delete(z.Series, key)
				}
			}
			for zuop > 0 {
				var zsnw string
				var ztls *WhisperDataItem
				zuop--
				zsnw, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					ztls = nil
				} else {
					if ztls == nil {
						ztls = new(WhisperDataItem)
					}
					bts, err = ztls.UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
				z.Series[zsnw] = ztls
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
func (z *WhisperRenderItem) Msgsize() (s int) {
	s = 1 + 11 + msgp.Uint32Size + 9 + msgp.Uint32Size + 6 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.Uint32Size + 3 + msgp.Uint32Size + 5 + msgp.Uint32Size + 7 + msgp.MapHeaderSize
	if z.Series != nil {
		for zsnw, ztls := range z.Series {
			_ = ztls
			s += msgp.StringPrefixSize + len(zsnw)
			if ztls == nil {
				s += msgp.NilSize
			} else {
				s += ztls.Msgsize()
			}
		}
	}
	return
}

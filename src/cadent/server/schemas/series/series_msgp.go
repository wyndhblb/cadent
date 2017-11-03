package series

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/schemas/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *AnyMetric) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Raw":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Raw = nil
			} else {
				if z.Raw == nil {
					z.Raw = new(RawMetric)
				}
				err = z.Raw.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Unprocessed":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Unprocessed = nil
			} else {
				if z.Unprocessed == nil {
					z.Unprocessed = new(UnProcessedMetric)
				}
				err = z.Unprocessed.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Single":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Single = nil
			} else {
				if z.Single == nil {
					z.Single = new(SingleMetric)
				}
				err = z.Single.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Series":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Series = nil
			} else {
				if z.Series == nil {
					z.Series = new(SeriesMetric)
				}
				err = z.Series.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "UidMetric":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.UidMetric = nil
			} else {
				if z.UidMetric == nil {
					z.UidMetric = new(UidMetric)
				}
				err = z.UidMetric.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Written":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Written = nil
			} else {
				if z.Written == nil {
					z.Written = new(MetricWritten)
				}
				err = z.Written.DecodeMsg(dc)
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
func (z *AnyMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Raw"
	err = en.Append(0x86, 0xa3, 0x52, 0x61, 0x77)
	if err != nil {
		return err
	}
	if z.Raw == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Raw.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Unprocessed"
	err = en.Append(0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	if err != nil {
		return err
	}
	if z.Unprocessed == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Unprocessed.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Single"
	err = en.Append(0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	if err != nil {
		return err
	}
	if z.Single == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Single.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Series"
	err = en.Append(0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	if z.Series == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Series.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "UidMetric"
	err = en.Append(0xa9, 0x55, 0x69, 0x64, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	if z.UidMetric == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.UidMetric.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Written"
	err = en.Append(0xa7, 0x57, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if err != nil {
		return err
	}
	if z.Written == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Written.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *AnyMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Raw"
	o = append(o, 0x86, 0xa3, 0x52, 0x61, 0x77)
	if z.Raw == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Raw.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Unprocessed"
	o = append(o, 0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	if z.Unprocessed == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Unprocessed.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Single"
	o = append(o, 0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	if z.Single == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Single.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Series"
	o = append(o, 0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	if z.Series == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Series.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "UidMetric"
	o = append(o, 0xa9, 0x55, 0x69, 0x64, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if z.UidMetric == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.UidMetric.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Written"
	o = append(o, 0xa7, 0x57, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if z.Written == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Written.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *AnyMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Raw":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Raw = nil
			} else {
				if z.Raw == nil {
					z.Raw = new(RawMetric)
				}
				bts, err = z.Raw.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Unprocessed":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Unprocessed = nil
			} else {
				if z.Unprocessed == nil {
					z.Unprocessed = new(UnProcessedMetric)
				}
				bts, err = z.Unprocessed.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Single":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Single = nil
			} else {
				if z.Single == nil {
					z.Single = new(SingleMetric)
				}
				bts, err = z.Single.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Series":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Series = nil
			} else {
				if z.Series == nil {
					z.Series = new(SeriesMetric)
				}
				bts, err = z.Series.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "UidMetric":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.UidMetric = nil
			} else {
				if z.UidMetric == nil {
					z.UidMetric = new(UidMetric)
				}
				bts, err = z.UidMetric.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Written":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Written = nil
			} else {
				if z.Written == nil {
					z.Written = new(MetricWritten)
				}
				bts, err = z.Written.UnmarshalMsg(bts)
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
func (z *AnyMetric) Msgsize() (s int) {
	s = 1 + 4
	if z.Raw == nil {
		s += msgp.NilSize
	} else {
		s += z.Raw.Msgsize()
	}
	s += 12
	if z.Unprocessed == nil {
		s += msgp.NilSize
	} else {
		s += z.Unprocessed.Msgsize()
	}
	s += 7
	if z.Single == nil {
		s += msgp.NilSize
	} else {
		s += z.Single.Msgsize()
	}
	s += 7
	if z.Series == nil {
		s += msgp.NilSize
	} else {
		s += z.Series.Msgsize()
	}
	s += 10
	if z.UidMetric == nil {
		s += msgp.NilSize
	} else {
		s += z.UidMetric.Msgsize()
	}
	s += 8
	if z.Written == nil {
		s += msgp.NilSize
	} else {
		s += z.Written.Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *AnyMetricList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
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
		case "Raw":
			var zlqf uint32
			zlqf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Raw) >= int(zlqf) {
				z.Raw = (z.Raw)[:zlqf]
			} else {
				z.Raw = make([]*RawMetric, zlqf)
			}
			for zbai := range z.Raw {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Raw[zbai] = nil
				} else {
					if z.Raw[zbai] == nil {
						z.Raw[zbai] = new(RawMetric)
					}
					err = z.Raw[zbai].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Unprocessed":
			var zdaf uint32
			zdaf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Unprocessed) >= int(zdaf) {
				z.Unprocessed = (z.Unprocessed)[:zdaf]
			} else {
				z.Unprocessed = make([]*UnProcessedMetric, zdaf)
			}
			for zcmr := range z.Unprocessed {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Unprocessed[zcmr] = nil
				} else {
					if z.Unprocessed[zcmr] == nil {
						z.Unprocessed[zcmr] = new(UnProcessedMetric)
					}
					err = z.Unprocessed[zcmr].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Single":
			var zpks uint32
			zpks, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Single) >= int(zpks) {
				z.Single = (z.Single)[:zpks]
			} else {
				z.Single = make([]*SingleMetric, zpks)
			}
			for zajw := range z.Single {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Single[zajw] = nil
				} else {
					if z.Single[zajw] == nil {
						z.Single[zajw] = new(SingleMetric)
					}
					err = z.Single[zajw].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Series":
			var zjfb uint32
			zjfb, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Series) >= int(zjfb) {
				z.Series = (z.Series)[:zjfb]
			} else {
				z.Series = make([]*SeriesMetric, zjfb)
			}
			for zwht := range z.Series {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Series[zwht] = nil
				} else {
					if z.Series[zwht] == nil {
						z.Series[zwht] = new(SeriesMetric)
					}
					err = z.Series[zwht].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "UidMetric":
			var zcxo uint32
			zcxo, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.UidMetric) >= int(zcxo) {
				z.UidMetric = (z.UidMetric)[:zcxo]
			} else {
				z.UidMetric = make([]*UidMetric, zcxo)
			}
			for zhct := range z.UidMetric {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.UidMetric[zhct] = nil
				} else {
					if z.UidMetric[zhct] == nil {
						z.UidMetric[zhct] = new(UidMetric)
					}
					err = z.UidMetric[zhct].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Written":
			var zeff uint32
			zeff, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Written) >= int(zeff) {
				z.Written = (z.Written)[:zeff]
			} else {
				z.Written = make([]*MetricWritten, zeff)
			}
			for zcua := range z.Written {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Written[zcua] = nil
				} else {
					if z.Written[zcua] == nil {
						z.Written[zcua] = new(MetricWritten)
					}
					err = z.Written[zcua].DecodeMsg(dc)
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
func (z *AnyMetricList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Raw"
	err = en.Append(0x86, 0xa3, 0x52, 0x61, 0x77)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Raw)))
	if err != nil {
		return
	}
	for zbai := range z.Raw {
		if z.Raw[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Raw[zbai].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Unprocessed"
	err = en.Append(0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Unprocessed)))
	if err != nil {
		return
	}
	for zcmr := range z.Unprocessed {
		if z.Unprocessed[zcmr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Unprocessed[zcmr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Single"
	err = en.Append(0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Single)))
	if err != nil {
		return
	}
	for zajw := range z.Single {
		if z.Single[zajw] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Single[zajw].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Series"
	err = en.Append(0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Series)))
	if err != nil {
		return
	}
	for zwht := range z.Series {
		if z.Series[zwht] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Series[zwht].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "UidMetric"
	err = en.Append(0xa9, 0x55, 0x69, 0x64, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.UidMetric)))
	if err != nil {
		return
	}
	for zhct := range z.UidMetric {
		if z.UidMetric[zhct] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.UidMetric[zhct].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Written"
	err = en.Append(0xa7, 0x57, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Written)))
	if err != nil {
		return
	}
	for zcua := range z.Written {
		if z.Written[zcua] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Written[zcua].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *AnyMetricList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Raw"
	o = append(o, 0x86, 0xa3, 0x52, 0x61, 0x77)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Raw)))
	for zbai := range z.Raw {
		if z.Raw[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Raw[zbai].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Unprocessed"
	o = append(o, 0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Unprocessed)))
	for zcmr := range z.Unprocessed {
		if z.Unprocessed[zcmr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Unprocessed[zcmr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Single"
	o = append(o, 0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Single)))
	for zajw := range z.Single {
		if z.Single[zajw] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Single[zajw].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Series"
	o = append(o, 0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Series)))
	for zwht := range z.Series {
		if z.Series[zwht] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Series[zwht].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "UidMetric"
	o = append(o, 0xa9, 0x55, 0x69, 0x64, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendArrayHeader(o, uint32(len(z.UidMetric)))
	for zhct := range z.UidMetric {
		if z.UidMetric[zhct] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.UidMetric[zhct].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Written"
	o = append(o, 0xa7, 0x57, 0x72, 0x69, 0x74, 0x74, 0x65, 0x6e)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Written)))
	for zcua := range z.Written {
		if z.Written[zcua] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Written[zcua].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *AnyMetricList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
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
		case "Raw":
			var zxpk uint32
			zxpk, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Raw) >= int(zxpk) {
				z.Raw = (z.Raw)[:zxpk]
			} else {
				z.Raw = make([]*RawMetric, zxpk)
			}
			for zbai := range z.Raw {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Raw[zbai] = nil
				} else {
					if z.Raw[zbai] == nil {
						z.Raw[zbai] = new(RawMetric)
					}
					bts, err = z.Raw[zbai].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Unprocessed":
			var zdnj uint32
			zdnj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Unprocessed) >= int(zdnj) {
				z.Unprocessed = (z.Unprocessed)[:zdnj]
			} else {
				z.Unprocessed = make([]*UnProcessedMetric, zdnj)
			}
			for zcmr := range z.Unprocessed {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Unprocessed[zcmr] = nil
				} else {
					if z.Unprocessed[zcmr] == nil {
						z.Unprocessed[zcmr] = new(UnProcessedMetric)
					}
					bts, err = z.Unprocessed[zcmr].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Single":
			var zobc uint32
			zobc, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Single) >= int(zobc) {
				z.Single = (z.Single)[:zobc]
			} else {
				z.Single = make([]*SingleMetric, zobc)
			}
			for zajw := range z.Single {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Single[zajw] = nil
				} else {
					if z.Single[zajw] == nil {
						z.Single[zajw] = new(SingleMetric)
					}
					bts, err = z.Single[zajw].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Series":
			var zsnv uint32
			zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Series) >= int(zsnv) {
				z.Series = (z.Series)[:zsnv]
			} else {
				z.Series = make([]*SeriesMetric, zsnv)
			}
			for zwht := range z.Series {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Series[zwht] = nil
				} else {
					if z.Series[zwht] == nil {
						z.Series[zwht] = new(SeriesMetric)
					}
					bts, err = z.Series[zwht].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "UidMetric":
			var zkgt uint32
			zkgt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.UidMetric) >= int(zkgt) {
				z.UidMetric = (z.UidMetric)[:zkgt]
			} else {
				z.UidMetric = make([]*UidMetric, zkgt)
			}
			for zhct := range z.UidMetric {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.UidMetric[zhct] = nil
				} else {
					if z.UidMetric[zhct] == nil {
						z.UidMetric[zhct] = new(UidMetric)
					}
					bts, err = z.UidMetric[zhct].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Written":
			var zema uint32
			zema, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Written) >= int(zema) {
				z.Written = (z.Written)[:zema]
			} else {
				z.Written = make([]*MetricWritten, zema)
			}
			for zcua := range z.Written {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Written[zcua] = nil
				} else {
					if z.Written[zcua] == nil {
						z.Written[zcua] = new(MetricWritten)
					}
					bts, err = z.Written[zcua].UnmarshalMsg(bts)
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
func (z *AnyMetricList) Msgsize() (s int) {
	s = 1 + 4 + msgp.ArrayHeaderSize
	for zbai := range z.Raw {
		if z.Raw[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += z.Raw[zbai].Msgsize()
		}
	}
	s += 12 + msgp.ArrayHeaderSize
	for zcmr := range z.Unprocessed {
		if z.Unprocessed[zcmr] == nil {
			s += msgp.NilSize
		} else {
			s += z.Unprocessed[zcmr].Msgsize()
		}
	}
	s += 7 + msgp.ArrayHeaderSize
	for zajw := range z.Single {
		if z.Single[zajw] == nil {
			s += msgp.NilSize
		} else {
			s += z.Single[zajw].Msgsize()
		}
	}
	s += 7 + msgp.ArrayHeaderSize
	for zwht := range z.Series {
		if z.Series[zwht] == nil {
			s += msgp.NilSize
		} else {
			s += z.Series[zwht].Msgsize()
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zhct := range z.UidMetric {
		if z.UidMetric[zhct] == nil {
			s += msgp.NilSize
		} else {
			s += z.UidMetric[zhct].Msgsize()
		}
	}
	s += 8 + msgp.ArrayHeaderSize
	for zcua := range z.Written {
		if z.Written[zcua] == nil {
			s += msgp.NilSize
		} else {
			s += z.Written[zcua].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricName) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zqyh uint32
	zqyh, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zqyh > 0 {
		zqyh--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Tags":
			var zyzr uint32
			zyzr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zyzr) {
				z.Tags = (z.Tags)[:zyzr]
			} else {
				z.Tags = make([]*repr.Tag, zyzr)
			}
			for zpez := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zpez] = nil
				} else {
					if z.Tags[zpez] == nil {
						z.Tags[zpez] = new(repr.Tag)
					}
					err = z.Tags[zpez].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zywj uint32
			zywj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zywj) {
				z.MetaTags = (z.MetaTags)[:zywj]
			} else {
				z.MetaTags = make([]*repr.Tag, zywj)
			}
			for zqke := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zqke] = nil
				} else {
					if z.MetaTags[zqke] == nil {
						z.MetaTags[zqke] = new(repr.Tag)
					}
					err = z.MetaTags[zqke].DecodeMsg(dc)
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
func (z *MetricName) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "Metric"
	err = en.Append(0x85, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "Id"
	err = en.Append(0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
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
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zpez].EncodeMsg(en)
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
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zqke].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricName) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "Metric"
	o = append(o, 0x85, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Id"
	o = append(o, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zpez].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zqke].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricName) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zjpj uint32
	zjpj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zjpj > 0 {
		zjpj--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zzpf uint32
			zzpf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zzpf) {
				z.Tags = (z.Tags)[:zzpf]
			} else {
				z.Tags = make([]*repr.Tag, zzpf)
			}
			for zpez := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zpez] = nil
				} else {
					if z.Tags[zpez] == nil {
						z.Tags[zpez] = new(repr.Tag)
					}
					bts, err = z.Tags[zpez].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zrfe uint32
			zrfe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrfe) {
				z.MetaTags = (z.MetaTags)[:zrfe]
			} else {
				z.MetaTags = make([]*repr.Tag, zrfe)
			}
			for zqke := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zqke] = nil
				} else {
					if z.MetaTags[zqke] == nil {
						z.MetaTags[zqke] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zqke].UnmarshalMsg(bts)
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
func (z *MetricName) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Metric) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.ArrayHeaderSize
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zpez].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zqke].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricType) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zgmo uint32
	zgmo, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zgmo > 0 {
		zgmo--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Type":
			z.Type, err = dc.ReadString()
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
func (z MetricType) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Type"
	err = en.Append(0x81, 0xa4, 0x54, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Type)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricType) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Type"
	o = append(o, 0x81, 0xa4, 0x54, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricType) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var ztaf uint32
	ztaf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for ztaf > 0 {
		ztaf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Type":
			z.Type, bts, err = msgp.ReadStringBytes(bts)
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
func (z MetricType) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricValue) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zeth uint32
	zeth, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zeth > 0 {
		zeth--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
func (z *MetricValue) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Time"
	err = en.Append(0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
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
func (z *MetricValue) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Time"
	o = append(o, 0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
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
func (z *MetricValue) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zsbz uint32
	zsbz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zsbz > 0 {
		zsbz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
func (z *MetricValue) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricWritten) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zwel uint32
	zwel, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zwel > 0 {
		zwel--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "StartTime":
			z.StartTime, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "EndTime":
			z.EndTime, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "WriteTime":
			z.WriteTime, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Offset":
			z.Offset, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Partition":
			z.Partition, err = dc.ReadInt32()
			if err != nil {
				return
			}
		case "Topic":
			z.Topic, err = dc.ReadString()
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
		case "Tags":
			var zrbe uint32
			zrbe, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrbe) {
				z.Tags = (z.Tags)[:zrbe]
			} else {
				z.Tags = make([]*repr.Tag, zrbe)
			}
			for zrjx := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zrjx] = nil
				} else {
					if z.Tags[zrjx] == nil {
						z.Tags[zrjx] = new(repr.Tag)
					}
					err = z.Tags[zrjx].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zmfd uint32
			zmfd, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zmfd) {
				z.MetaTags = (z.MetaTags)[:zmfd]
			} else {
				z.MetaTags = make([]*repr.Tag, zmfd)
			}
			for zawn := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zawn] = nil
				} else {
					if z.MetaTags[zawn] == nil {
						z.MetaTags[zawn] = new(repr.Tag)
					}
					err = z.MetaTags[zawn].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *MetricWritten) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "Id"
	err = en.Append(0x8e, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "StartTime"
	err = en.Append(0xa9, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.StartTime)
	if err != nil {
		return
	}
	// write "EndTime"
	err = en.Append(0xa7, 0x45, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.EndTime)
	if err != nil {
		return
	}
	// write "WriteTime"
	err = en.Append(0xa9, 0x57, 0x72, 0x69, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.WriteTime)
	if err != nil {
		return
	}
	// write "Offset"
	err = en.Append(0xa6, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Offset)
	if err != nil {
		return
	}
	// write "Partition"
	err = en.Append(0xa9, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt32(z.Partition)
	if err != nil {
		return
	}
	// write "Topic"
	err = en.Append(0xa5, 0x54, 0x6f, 0x70, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Topic)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zrjx := range z.Tags {
		if z.Tags[zrjx] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zrjx].EncodeMsg(en)
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
	for zawn := range z.MetaTags {
		if z.MetaTags[zawn] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zawn].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricWritten) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "Id"
	o = append(o, 0x8e, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "StartTime"
	o = append(o, 0xa9, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.StartTime)
	// string "EndTime"
	o = append(o, 0xa7, 0x45, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.EndTime)
	// string "WriteTime"
	o = append(o, 0xa9, 0x57, 0x72, 0x69, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.WriteTime)
	// string "Offset"
	o = append(o, 0xa6, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74)
	o = msgp.AppendInt64(o, z.Offset)
	// string "Partition"
	o = append(o, 0xa9, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt32(o, z.Partition)
	// string "Topic"
	o = append(o, 0xa5, 0x54, 0x6f, 0x70, 0x69, 0x63)
	o = msgp.AppendString(o, z.Topic)
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zrjx := range z.Tags {
		if z.Tags[zrjx] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zrjx].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zawn := range z.MetaTags {
		if z.MetaTags[zawn] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zawn].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricWritten) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zzdc uint32
	zzdc, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zzdc > 0 {
		zzdc--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "StartTime":
			z.StartTime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "EndTime":
			z.EndTime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "WriteTime":
			z.WriteTime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Offset":
			z.Offset, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Partition":
			z.Partition, bts, err = msgp.ReadInt32Bytes(bts)
			if err != nil {
				return
			}
		case "Topic":
			z.Topic, bts, err = msgp.ReadStringBytes(bts)
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
		case "Tags":
			var zelx uint32
			zelx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zelx) {
				z.Tags = (z.Tags)[:zelx]
			} else {
				z.Tags = make([]*repr.Tag, zelx)
			}
			for zrjx := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zrjx] = nil
				} else {
					if z.Tags[zrjx] == nil {
						z.Tags[zrjx] = new(repr.Tag)
					}
					bts, err = z.Tags[zrjx].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zbal uint32
			zbal, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zbal) {
				z.MetaTags = (z.MetaTags)[:zbal]
			} else {
				z.MetaTags = make([]*repr.Tag, zbal)
			}
			for zawn := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zawn] = nil
				} else {
					if z.MetaTags[zawn] == nil {
						z.MetaTags[zawn] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zawn].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *MetricWritten) Msgsize() (s int) {
	s = 1 + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 7 + msgp.StringPrefixSize + len(z.Metric) + 10 + msgp.Int64Size + 8 + msgp.Int64Size + 10 + msgp.Int64Size + 7 + msgp.Int64Size + 10 + msgp.Int32Size + 6 + msgp.StringPrefixSize + len(z.Topic) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zrjx := range z.Tags {
		if z.Tags[zrjx] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zrjx].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zawn := range z.MetaTags {
		if z.MetaTags[zawn] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zawn].Msgsize()
		}
	}
	s += 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ztmt uint32
	ztmt, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ztmt > 0 {
		ztmt--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Value":
			z.Value, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Tags":
			var ztco uint32
			ztco, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(ztco) {
				z.Tags = (z.Tags)[:ztco]
			} else {
				z.Tags = make([]*repr.Tag, ztco)
			}
			for zjqz := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zjqz] = nil
				} else {
					if z.Tags[zjqz] == nil {
						z.Tags[zjqz] = new(repr.Tag)
					}
					err = z.Tags[zjqz].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zana uint32
			zana, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zana) {
				z.MetaTags = (z.MetaTags)[:zana]
			} else {
				z.MetaTags = make([]*repr.Tag, zana)
			}
			for zkct := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zkct] = nil
				} else {
					if z.MetaTags[zkct] == nil {
						z.MetaTags[zkct] = new(repr.Tag)
					}
					err = z.MetaTags[zkct].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *RawMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Time"
	err = en.Append(0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "Value"
	err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Value)
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
	for zjqz := range z.Tags {
		if z.Tags[zjqz] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zjqz].EncodeMsg(en)
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
	for zkct := range z.MetaTags {
		if z.MetaTags[zkct] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zkct].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Time"
	o = append(o, 0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendFloat64(o, z.Value)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zjqz := range z.Tags {
		if z.Tags[zjqz] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zjqz].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zkct := range z.MetaTags {
		if z.MetaTags[zkct] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zkct].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var ztyy uint32
	ztyy, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for ztyy > 0 {
		ztyy--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zinl uint32
			zinl, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zinl) {
				z.Tags = (z.Tags)[:zinl]
			} else {
				z.Tags = make([]*repr.Tag, zinl)
			}
			for zjqz := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zjqz] = nil
				} else {
					if z.Tags[zjqz] == nil {
						z.Tags[zjqz] = new(repr.Tag)
					}
					bts, err = z.Tags[zjqz].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zare uint32
			zare, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zare) {
				z.MetaTags = (z.MetaTags)[:zare]
			} else {
				z.MetaTags = make([]*repr.Tag, zare)
			}
			for zkct := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zkct] = nil
				} else {
					if z.MetaTags[zkct] == nil {
						z.MetaTags[zkct] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zkct].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *RawMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 6 + msgp.Float64Size + 5 + msgp.ArrayHeaderSize
	for zjqz := range z.Tags {
		if z.Tags[zjqz] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zjqz].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zkct := range z.MetaTags {
		if z.MetaTags[zkct] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zkct].Msgsize()
		}
	}
	s += 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawMetricList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zixj uint32
	zixj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zixj > 0 {
		zixj--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zrsc uint32
			zrsc, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.List) >= int(zrsc) {
				z.List = (z.List)[:zrsc]
			} else {
				z.List = make([]*RawMetric, zrsc)
			}
			for zljy := range z.List {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.List[zljy] = nil
				} else {
					if z.List[zljy] == nil {
						z.List[zljy] = new(RawMetric)
					}
					err = z.List[zljy].DecodeMsg(dc)
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
func (z *RawMetricList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "List"
	err = en.Append(0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.List)))
	if err != nil {
		return
	}
	for zljy := range z.List {
		if z.List[zljy] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.List[zljy].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawMetricList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "List"
	o = append(o, 0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	o = msgp.AppendArrayHeader(o, uint32(len(z.List)))
	for zljy := range z.List {
		if z.List[zljy] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.List[zljy].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawMetricList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zctn uint32
	zctn, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zctn > 0 {
		zctn--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zswy uint32
			zswy, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.List) >= int(zswy) {
				z.List = (z.List)[:zswy]
			} else {
				z.List = make([]*RawMetric, zswy)
			}
			for zljy := range z.List {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.List[zljy] = nil
				} else {
					if z.List[zljy] == nil {
						z.List[zljy] = new(RawMetric)
					}
					bts, err = z.List[zljy].UnmarshalMsg(bts)
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
func (z *RawMetricList) Msgsize() (s int) {
	s = 1 + 5 + msgp.ArrayHeaderSize
	for zljy := range z.List {
		if z.List[zljy] == nil {
			s += msgp.NilSize
		} else {
			s += z.List[zljy].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zsvm uint32
	zsvm, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zsvm > 0 {
		zsvm--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Encoding":
			z.Encoding, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Data":
			z.Data, err = dc.ReadBytes(z.Data)
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
		case "Tags":
			var zaoz uint32
			zaoz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zaoz) {
				z.Tags = (z.Tags)[:zaoz]
			} else {
				z.Tags = make([]*repr.Tag, zaoz)
			}
			for znsg := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[znsg] = nil
				} else {
					if z.Tags[znsg] == nil {
						z.Tags[znsg] = new(repr.Tag)
					}
					err = z.Tags[znsg].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zfzb uint32
			zfzb, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zfzb) {
				z.MetaTags = (z.MetaTags)[:zfzb]
			} else {
				z.MetaTags = make([]*repr.Tag, zfzb)
			}
			for zrus := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zrus] = nil
				} else {
					if z.MetaTags[zrus] == nil {
						z.MetaTags[zrus] = new(repr.Tag)
					}
					err = z.MetaTags[zrus].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *SeriesMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 11
	// write "Id"
	err = en.Append(0x8b, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
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
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "Encoding"
	err = en.Append(0xa8, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Encoding)
	if err != nil {
		return
	}
	// write "Data"
	err = en.Append(0xa4, 0x44, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteBytes(z.Data)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for znsg := range z.Tags {
		if z.Tags[znsg] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[znsg].EncodeMsg(en)
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
	for zrus := range z.MetaTags {
		if z.MetaTags[zrus] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zrus].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SeriesMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 11
	// string "Id"
	o = append(o, 0x8b, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Time"
	o = append(o, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Encoding"
	o = append(o, 0xa8, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	o = msgp.AppendString(o, z.Encoding)
	// string "Data"
	o = append(o, 0xa4, 0x44, 0x61, 0x74, 0x61)
	o = msgp.AppendBytes(o, z.Data)
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for znsg := range z.Tags {
		if z.Tags[znsg] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[znsg].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zrus := range z.MetaTags {
		if z.MetaTags[zrus] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zrus].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zsbo uint32
	zsbo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zsbo > 0 {
		zsbo--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Encoding":
			z.Encoding, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Data":
			z.Data, bts, err = msgp.ReadBytesBytes(bts, z.Data)
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
		case "Tags":
			var zjif uint32
			zjif, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zjif) {
				z.Tags = (z.Tags)[:zjif]
			} else {
				z.Tags = make([]*repr.Tag, zjif)
			}
			for znsg := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[znsg] = nil
				} else {
					if z.Tags[znsg] == nil {
						z.Tags[znsg] = new(repr.Tag)
					}
					bts, err = z.Tags[znsg].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zqgz uint32
			zqgz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zqgz) {
				z.MetaTags = (z.MetaTags)[:zqgz]
			} else {
				z.MetaTags = make([]*repr.Tag, zqgz)
			}
			for zrus := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zrus] = nil
				} else {
					if z.MetaTags[zrus] == nil {
						z.MetaTags[zrus] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zrus].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *SeriesMetric) Msgsize() (s int) {
	s = 1 + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 9 + msgp.StringPrefixSize + len(z.Encoding) + 5 + msgp.BytesPrefixSize + len(z.Data) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for znsg := range z.Tags {
		if z.Tags[znsg] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[znsg].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zrus := range z.MetaTags {
		if z.MetaTags[zrus] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zrus].Msgsize()
		}
	}
	s += 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
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
		case "Tags":
			var zigk uint32
			zigk, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zigk) {
				z.Tags = (z.Tags)[:zigk]
			} else {
				z.Tags = make([]*repr.Tag, zigk)
			}
			for zsnw := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zsnw] = nil
				} else {
					if z.Tags[zsnw] == nil {
						z.Tags[zsnw] = new(repr.Tag)
					}
					err = z.Tags[zsnw].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zopb uint32
			zopb, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zopb) {
				z.MetaTags = (z.MetaTags)[:zopb]
			} else {
				z.MetaTags = make([]*repr.Tag, zopb)
			}
			for ztls := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[ztls] = nil
				} else {
					if z.MetaTags[ztls] == nil {
						z.MetaTags[ztls] = new(repr.Tag)
					}
					err = z.MetaTags[ztls].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *SingleMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "Id"
	err = en.Append(0x8e, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
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
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zsnw := range z.Tags {
		if z.Tags[zsnw] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zsnw].EncodeMsg(en)
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
	for ztls := range z.MetaTags {
		if z.MetaTags[ztls] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[ztls].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SingleMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "Id"
	o = append(o, 0x8e, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Time"
	o = append(o, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
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
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zsnw := range z.Tags {
		if z.Tags[zsnw] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zsnw].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for ztls := range z.MetaTags {
		if z.MetaTags[ztls] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[ztls].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zuop uint32
	zuop, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zuop > 0 {
		zuop--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
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
		case "Tags":
			var zedl uint32
			zedl, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zedl) {
				z.Tags = (z.Tags)[:zedl]
			} else {
				z.Tags = make([]*repr.Tag, zedl)
			}
			for zsnw := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zsnw] = nil
				} else {
					if z.Tags[zsnw] == nil {
						z.Tags[zsnw] = new(repr.Tag)
					}
					bts, err = z.Tags[zsnw].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zupd uint32
			zupd, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zupd) {
				z.MetaTags = (z.MetaTags)[:zupd]
			} else {
				z.MetaTags = make([]*repr.Tag, zupd)
			}
			for ztls := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[ztls] = nil
				} else {
					if z.MetaTags[ztls] == nil {
						z.MetaTags[ztls] = new(repr.Tag)
					}
					bts, err = z.MetaTags[ztls].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *SingleMetric) Msgsize() (s int) {
	s = 1 + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zsnw := range z.Tags {
		if z.Tags[zsnw] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zsnw].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for ztls := range z.MetaTags {
		if z.MetaTags[ztls] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[ztls].Msgsize()
		}
	}
	s += 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetricList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zrvj uint32
	zrvj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zrvj > 0 {
		zrvj--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zarz uint32
			zarz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.List) >= int(zarz) {
				z.List = (z.List)[:zarz]
			} else {
				z.List = make([]*SingleMetric, zarz)
			}
			for zome := range z.List {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.List[zome] = nil
				} else {
					if z.List[zome] == nil {
						z.List[zome] = new(SingleMetric)
					}
					err = z.List[zome].DecodeMsg(dc)
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
func (z *SingleMetricList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "List"
	err = en.Append(0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.List)))
	if err != nil {
		return
	}
	for zome := range z.List {
		if z.List[zome] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.List[zome].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SingleMetricList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "List"
	o = append(o, 0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	o = msgp.AppendArrayHeader(o, uint32(len(z.List)))
	for zome := range z.List {
		if z.List[zome] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.List[zome].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetricList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zknt uint32
	zknt, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zknt > 0 {
		zknt--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zxye uint32
			zxye, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.List) >= int(zxye) {
				z.List = (z.List)[:zxye]
			} else {
				z.List = make([]*SingleMetric, zxye)
			}
			for zome := range z.List {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.List[zome] = nil
				} else {
					if z.List[zome] == nil {
						z.List[zome] = new(SingleMetric)
					}
					bts, err = z.List[zome].UnmarshalMsg(bts)
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
func (z *SingleMetricList) Msgsize() (s int) {
	s = 1 + 5 + msgp.ArrayHeaderSize
	for zome := range z.List {
		if z.List[zome] == nil {
			s += msgp.NilSize
		} else {
			s += z.List[zome].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UidMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zucw uint32
	zucw, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zucw > 0 {
		zucw--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
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
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *UidMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "Uid"
	err = en.Append(0x88, 0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
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
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *UidMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "Uid"
	o = append(o, 0x88, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
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
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UidMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zlsx uint32
	zlsx, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zlsx > 0 {
		zlsx--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
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
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *UidMetric) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UidMetricList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zrao uint32
	zrao, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zrao > 0 {
		zrao--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zmbt uint32
			zmbt, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.List) >= int(zmbt) {
				z.List = (z.List)[:zmbt]
			} else {
				z.List = make([]*UidMetricList, zmbt)
			}
			for zbgy := range z.List {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.List[zbgy] = nil
				} else {
					if z.List[zbgy] == nil {
						z.List[zbgy] = new(UidMetricList)
					}
					err = z.List[zbgy].DecodeMsg(dc)
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
func (z *UidMetricList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "List"
	err = en.Append(0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.List)))
	if err != nil {
		return
	}
	for zbgy := range z.List {
		if z.List[zbgy] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.List[zbgy].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *UidMetricList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "List"
	o = append(o, 0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	o = msgp.AppendArrayHeader(o, uint32(len(z.List)))
	for zbgy := range z.List {
		if z.List[zbgy] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.List[zbgy].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UidMetricList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zvls uint32
	zvls, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zvls > 0 {
		zvls--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zjfj uint32
			zjfj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.List) >= int(zjfj) {
				z.List = (z.List)[:zjfj]
			} else {
				z.List = make([]*UidMetricList, zjfj)
			}
			for zbgy := range z.List {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.List[zbgy] = nil
				} else {
					if z.List[zbgy] == nil {
						z.List[zbgy] = new(UidMetricList)
					}
					bts, err = z.List[zbgy].UnmarshalMsg(bts)
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
func (z *UidMetricList) Msgsize() (s int) {
	s = 1 + 5 + msgp.ArrayHeaderSize
	for zbgy := range z.List {
		if z.List[zbgy] == nil {
			s += msgp.NilSize
		} else {
			s += z.List[zbgy].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zsym uint32
	zsym, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zsym > 0 {
		zsym--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
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
		case "Tags":
			var zgeu uint32
			zgeu, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zgeu) {
				z.Tags = (z.Tags)[:zgeu]
			} else {
				z.Tags = make([]*repr.Tag, zgeu)
			}
			for zzak := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zzak] = nil
				} else {
					if z.Tags[zzak] == nil {
						z.Tags[zzak] = new(repr.Tag)
					}
					err = z.Tags[zzak].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zdtr uint32
			zdtr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zdtr) {
				z.MetaTags = (z.MetaTags)[:zdtr]
			} else {
				z.MetaTags = make([]*repr.Tag, zdtr)
			}
			for zbtz := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zbtz] = nil
				} else {
					if z.MetaTags[zbtz] == nil {
						z.MetaTags[zbtz] = new(repr.Tag)
					}
					err = z.MetaTags[zbtz].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, err = dc.ReadInt64()
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
func (z *UnProcessedMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "Time"
	err = en.Append(0x8a, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zzak].EncodeMsg(en)
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
	for zbtz := range z.MetaTags {
		if z.MetaTags[zbtz] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zbtz].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "SentTime"
	err = en.Append(0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.SentTime)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *UnProcessedMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "Time"
	o = append(o, 0x8a, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
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
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zzak].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zbtz := range z.MetaTags {
		if z.MetaTags[zbtz] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zbtz].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "SentTime"
	o = append(o, 0xa8, 0x53, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.SentTime)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zzqm uint32
	zzqm, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zzqm > 0 {
		zzqm--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
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
		case "Tags":
			var zdqi uint32
			zdqi, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zdqi) {
				z.Tags = (z.Tags)[:zdqi]
			} else {
				z.Tags = make([]*repr.Tag, zdqi)
			}
			for zzak := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zzak] = nil
				} else {
					if z.Tags[zzak] == nil {
						z.Tags[zzak] = new(repr.Tag)
					}
					bts, err = z.Tags[zzak].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zyco uint32
			zyco, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zyco) {
				z.MetaTags = (z.MetaTags)[:zyco]
			} else {
				z.MetaTags = make([]*repr.Tag, zyco)
			}
			for zbtz := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zbtz] = nil
				} else {
					if z.MetaTags[zbtz] == nil {
						z.MetaTags[zbtz] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zbtz].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "SentTime":
			z.SentTime, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *UnProcessedMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 5 + msgp.ArrayHeaderSize
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zzak].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zbtz := range z.MetaTags {
		if z.MetaTags[zbtz] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zbtz].Msgsize()
		}
	}
	s += 9 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetricList) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zovg uint32
	zovg, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zovg > 0 {
		zovg--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zsey uint32
			zsey, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.List) >= int(zsey) {
				z.List = (z.List)[:zsey]
			} else {
				z.List = make([]*UnProcessedMetric, zsey)
			}
			for zhgh := range z.List {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.List[zhgh] = nil
				} else {
					if z.List[zhgh] == nil {
						z.List[zhgh] = new(UnProcessedMetric)
					}
					err = z.List[zhgh].DecodeMsg(dc)
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
func (z *UnProcessedMetricList) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "List"
	err = en.Append(0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.List)))
	if err != nil {
		return
	}
	for zhgh := range z.List {
		if z.List[zhgh] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.List[zhgh].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *UnProcessedMetricList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "List"
	o = append(o, 0x81, 0xa4, 0x4c, 0x69, 0x73, 0x74)
	o = msgp.AppendArrayHeader(o, uint32(len(z.List)))
	for zhgh := range z.List {
		if z.List[zhgh] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.List[zhgh].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetricList) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcjp uint32
	zcjp, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcjp > 0 {
		zcjp--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "List":
			var zjhy uint32
			zjhy, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.List) >= int(zjhy) {
				z.List = (z.List)[:zjhy]
			} else {
				z.List = make([]*UnProcessedMetric, zjhy)
			}
			for zhgh := range z.List {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.List[zhgh] = nil
				} else {
					if z.List[zhgh] == nil {
						z.List[zhgh] = new(UnProcessedMetric)
					}
					bts, err = z.List[zhgh].UnmarshalMsg(bts)
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
func (z *UnProcessedMetricList) Msgsize() (s int) {
	s = 1 + 5 + msgp.ArrayHeaderSize
	for zhgh := range z.List {
		if z.List[zhgh] == nil {
			s += msgp.NilSize
		} else {
			s += z.List[zhgh].Msgsize()
		}
	}
	return
}

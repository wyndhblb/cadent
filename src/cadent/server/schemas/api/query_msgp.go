package api

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/schemas/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *DiscoverHost) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zbai uint32
	zbai, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zbai > 0 {
		zbai--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "AdvertiseName":
			z.AdvertiseName, err = dc.ReadString()
			if err != nil {
				return
			}
		case "AdvertiseUrl":
			z.AdvertiseUrl, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Grpchost":
			z.Grpchost, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Host":
			z.Host, err = dc.ReadString()
			if err != nil {
				return
			}
		case "HostApiUrl":
			z.HostApiUrl, err = dc.ReadString()
			if err != nil {
				return
			}
		case "StartTime":
			z.StartTime, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Resolutions":
			var zcmr uint32
			zcmr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Resolutions) >= int(zcmr) {
				z.Resolutions = (z.Resolutions)[:zcmr]
			} else {
				z.Resolutions = make([]*Resolution, zcmr)
			}
			for zxvk := range z.Resolutions {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Resolutions[zxvk] = nil
				} else {
					if z.Resolutions[zxvk] == nil {
						z.Resolutions[zxvk] = new(Resolution)
					}
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
						case "Resolution":
							z.Resolutions[zxvk].Resolution, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "Ttl":
							z.Resolutions[zxvk].Ttl, err = dc.ReadUint32()
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
		case "IsApi":
			z.IsApi, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "IsWriter":
			z.IsWriter, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "IsReader":
			z.IsReader, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "IsHasher":
			z.IsHasher, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "IsTCPapi":
			z.IsTCPapi, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "IsgRPC":
			z.IsgRPC, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "Tags":
			var zwht uint32
			zwht, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zwht) {
				z.Tags = (z.Tags)[:zwht]
			} else {
				z.Tags = make([]*repr.Tag, zwht)
			}
			for zbzg := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zbzg] = nil
				} else {
					if z.Tags[zbzg] == nil {
						z.Tags[zbzg] = new(repr.Tag)
					}
					err = z.Tags[zbzg].DecodeMsg(dc)
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
func (z *DiscoverHost) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "AdvertiseName"
	err = en.Append(0x8e, 0xad, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x4e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.AdvertiseName)
	if err != nil {
		return
	}
	// write "AdvertiseUrl"
	err = en.Append(0xac, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x55, 0x72, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteString(z.AdvertiseUrl)
	if err != nil {
		return
	}
	// write "Grpchost"
	err = en.Append(0xa8, 0x47, 0x72, 0x70, 0x63, 0x68, 0x6f, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Grpchost)
	if err != nil {
		return
	}
	// write "Host"
	err = en.Append(0xa4, 0x48, 0x6f, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Host)
	if err != nil {
		return
	}
	// write "HostApiUrl"
	err = en.Append(0xaa, 0x48, 0x6f, 0x73, 0x74, 0x41, 0x70, 0x69, 0x55, 0x72, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteString(z.HostApiUrl)
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
	// write "Resolutions"
	err = en.Append(0xab, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Resolutions)))
	if err != nil {
		return
	}
	for zxvk := range z.Resolutions {
		if z.Resolutions[zxvk] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "Resolution"
			err = en.Append(0x82, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Resolutions[zxvk].Resolution)
			if err != nil {
				return
			}
			// write "Ttl"
			err = en.Append(0xa3, 0x54, 0x74, 0x6c)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Resolutions[zxvk].Ttl)
			if err != nil {
				return
			}
		}
	}
	// write "IsApi"
	err = en.Append(0xa5, 0x49, 0x73, 0x41, 0x70, 0x69)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsApi)
	if err != nil {
		return
	}
	// write "IsWriter"
	err = en.Append(0xa8, 0x49, 0x73, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsWriter)
	if err != nil {
		return
	}
	// write "IsReader"
	err = en.Append(0xa8, 0x49, 0x73, 0x52, 0x65, 0x61, 0x64, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsReader)
	if err != nil {
		return
	}
	// write "IsHasher"
	err = en.Append(0xa8, 0x49, 0x73, 0x48, 0x61, 0x73, 0x68, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsHasher)
	if err != nil {
		return
	}
	// write "IsTCPapi"
	err = en.Append(0xa8, 0x49, 0x73, 0x54, 0x43, 0x50, 0x61, 0x70, 0x69)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsTCPapi)
	if err != nil {
		return
	}
	// write "IsgRPC"
	err = en.Append(0xa6, 0x49, 0x73, 0x67, 0x52, 0x50, 0x43)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.IsgRPC)
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
	for zbzg := range z.Tags {
		if z.Tags[zbzg] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zbzg].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DiscoverHost) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "AdvertiseName"
	o = append(o, 0x8e, 0xad, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x4e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.AdvertiseName)
	// string "AdvertiseUrl"
	o = append(o, 0xac, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x55, 0x72, 0x6c)
	o = msgp.AppendString(o, z.AdvertiseUrl)
	// string "Grpchost"
	o = append(o, 0xa8, 0x47, 0x72, 0x70, 0x63, 0x68, 0x6f, 0x73, 0x74)
	o = msgp.AppendString(o, z.Grpchost)
	// string "Host"
	o = append(o, 0xa4, 0x48, 0x6f, 0x73, 0x74)
	o = msgp.AppendString(o, z.Host)
	// string "HostApiUrl"
	o = append(o, 0xaa, 0x48, 0x6f, 0x73, 0x74, 0x41, 0x70, 0x69, 0x55, 0x72, 0x6c)
	o = msgp.AppendString(o, z.HostApiUrl)
	// string "StartTime"
	o = append(o, 0xa9, 0x53, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.StartTime)
	// string "Resolutions"
	o = append(o, 0xab, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Resolutions)))
	for zxvk := range z.Resolutions {
		if z.Resolutions[zxvk] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "Resolution"
			o = append(o, 0x82, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
			o = msgp.AppendUint32(o, z.Resolutions[zxvk].Resolution)
			// string "Ttl"
			o = append(o, 0xa3, 0x54, 0x74, 0x6c)
			o = msgp.AppendUint32(o, z.Resolutions[zxvk].Ttl)
		}
	}
	// string "IsApi"
	o = append(o, 0xa5, 0x49, 0x73, 0x41, 0x70, 0x69)
	o = msgp.AppendBool(o, z.IsApi)
	// string "IsWriter"
	o = append(o, 0xa8, 0x49, 0x73, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72)
	o = msgp.AppendBool(o, z.IsWriter)
	// string "IsReader"
	o = append(o, 0xa8, 0x49, 0x73, 0x52, 0x65, 0x61, 0x64, 0x65, 0x72)
	o = msgp.AppendBool(o, z.IsReader)
	// string "IsHasher"
	o = append(o, 0xa8, 0x49, 0x73, 0x48, 0x61, 0x73, 0x68, 0x65, 0x72)
	o = msgp.AppendBool(o, z.IsHasher)
	// string "IsTCPapi"
	o = append(o, 0xa8, 0x49, 0x73, 0x54, 0x43, 0x50, 0x61, 0x70, 0x69)
	o = msgp.AppendBool(o, z.IsTCPapi)
	// string "IsgRPC"
	o = append(o, 0xa6, 0x49, 0x73, 0x67, 0x52, 0x50, 0x43)
	o = msgp.AppendBool(o, z.IsgRPC)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zbzg := range z.Tags {
		if z.Tags[zbzg] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zbzg].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DiscoverHost) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "AdvertiseName":
			z.AdvertiseName, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "AdvertiseUrl":
			z.AdvertiseUrl, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Grpchost":
			z.Grpchost, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Host":
			z.Host, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "HostApiUrl":
			z.HostApiUrl, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "StartTime":
			z.StartTime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Resolutions":
			var zcua uint32
			zcua, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Resolutions) >= int(zcua) {
				z.Resolutions = (z.Resolutions)[:zcua]
			} else {
				z.Resolutions = make([]*Resolution, zcua)
			}
			for zxvk := range z.Resolutions {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Resolutions[zxvk] = nil
				} else {
					if z.Resolutions[zxvk] == nil {
						z.Resolutions[zxvk] = new(Resolution)
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
						case "Resolution":
							z.Resolutions[zxvk].Resolution, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "Ttl":
							z.Resolutions[zxvk].Ttl, bts, err = msgp.ReadUint32Bytes(bts)
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
		case "IsApi":
			z.IsApi, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "IsWriter":
			z.IsWriter, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "IsReader":
			z.IsReader, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "IsHasher":
			z.IsHasher, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "IsTCPapi":
			z.IsTCPapi, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "IsgRPC":
			z.IsgRPC, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zlqf uint32
			zlqf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zlqf) {
				z.Tags = (z.Tags)[:zlqf]
			} else {
				z.Tags = make([]*repr.Tag, zlqf)
			}
			for zbzg := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zbzg] = nil
				} else {
					if z.Tags[zbzg] == nil {
						z.Tags[zbzg] = new(repr.Tag)
					}
					bts, err = z.Tags[zbzg].UnmarshalMsg(bts)
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
func (z *DiscoverHost) Msgsize() (s int) {
	s = 1 + 14 + msgp.StringPrefixSize + len(z.AdvertiseName) + 13 + msgp.StringPrefixSize + len(z.AdvertiseUrl) + 9 + msgp.StringPrefixSize + len(z.Grpchost) + 5 + msgp.StringPrefixSize + len(z.Host) + 11 + msgp.StringPrefixSize + len(z.HostApiUrl) + 10 + msgp.Int64Size + 12 + msgp.ArrayHeaderSize
	for zxvk := range z.Resolutions {
		if z.Resolutions[zxvk] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size
		}
	}
	s += 6 + msgp.BoolSize + 9 + msgp.BoolSize + 9 + msgp.BoolSize + 9 + msgp.BoolSize + 9 + msgp.BoolSize + 7 + msgp.BoolSize + 5 + msgp.ArrayHeaderSize
	for zbzg := range z.Tags {
		if z.Tags[zbzg] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zbzg].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *DiscoverHosts) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zpks uint32
	zpks, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zpks > 0 {
		zpks--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Hosts":
			var zjfb uint32
			zjfb, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Hosts) >= int(zjfb) {
				z.Hosts = (z.Hosts)[:zjfb]
			} else {
				z.Hosts = make([]*DiscoverHost, zjfb)
			}
			for zdaf := range z.Hosts {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Hosts[zdaf] = nil
				} else {
					if z.Hosts[zdaf] == nil {
						z.Hosts[zdaf] = new(DiscoverHost)
					}
					err = z.Hosts[zdaf].DecodeMsg(dc)
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
func (z *DiscoverHosts) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Hosts"
	err = en.Append(0x81, 0xa5, 0x48, 0x6f, 0x73, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Hosts)))
	if err != nil {
		return
	}
	for zdaf := range z.Hosts {
		if z.Hosts[zdaf] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Hosts[zdaf].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DiscoverHosts) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Hosts"
	o = append(o, 0x81, 0xa5, 0x48, 0x6f, 0x73, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Hosts)))
	for zdaf := range z.Hosts {
		if z.Hosts[zdaf] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Hosts[zdaf].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DiscoverHosts) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Hosts":
			var zeff uint32
			zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Hosts) >= int(zeff) {
				z.Hosts = (z.Hosts)[:zeff]
			} else {
				z.Hosts = make([]*DiscoverHost, zeff)
			}
			for zdaf := range z.Hosts {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Hosts[zdaf] = nil
				} else {
					if z.Hosts[zdaf] == nil {
						z.Hosts[zdaf] = new(DiscoverHost)
					}
					bts, err = z.Hosts[zdaf].UnmarshalMsg(bts)
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
func (z *DiscoverHosts) Msgsize() (s int) {
	s = 1 + 6 + msgp.ArrayHeaderSize
	for zdaf := range z.Hosts {
		if z.Hosts[zdaf] == nil {
			s += msgp.NilSize
		} else {
			s += z.Hosts[zdaf].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *DiscoveryQuery) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxpk uint32
	zxpk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxpk > 0 {
		zxpk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Host":
			z.Host, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Tags":
			var zdnj uint32
			zdnj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zdnj) {
				z.Tags = (z.Tags)[:zdnj]
			} else {
				z.Tags = make([]*repr.Tag, zdnj)
			}
			for zrsw := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zrsw] = nil
				} else {
					if z.Tags[zrsw] == nil {
						z.Tags[zrsw] = new(repr.Tag)
					}
					err = z.Tags[zrsw].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Format":
			z.Format, err = dc.ReadString()
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
func (z *DiscoveryQuery) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "Host"
	err = en.Append(0x83, 0xa4, 0x48, 0x6f, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Host)
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
	for zrsw := range z.Tags {
		if z.Tags[zrsw] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zrsw].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Format"
	err = en.Append(0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Format)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DiscoveryQuery) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Host"
	o = append(o, 0x83, 0xa4, 0x48, 0x6f, 0x73, 0x74)
	o = msgp.AppendString(o, z.Host)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zrsw := range z.Tags {
		if z.Tags[zrsw] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zrsw].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Format"
	o = append(o, 0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	o = msgp.AppendString(o, z.Format)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DiscoveryQuery) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zobc uint32
	zobc, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zobc > 0 {
		zobc--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Host":
			z.Host, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zsnv uint32
			zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zsnv) {
				z.Tags = (z.Tags)[:zsnv]
			} else {
				z.Tags = make([]*repr.Tag, zsnv)
			}
			for zrsw := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zrsw] = nil
				} else {
					if z.Tags[zrsw] == nil {
						z.Tags[zrsw] = new(repr.Tag)
					}
					bts, err = z.Tags[zrsw].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Format":
			z.Format, bts, err = msgp.ReadStringBytes(bts)
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
func (z *DiscoveryQuery) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Host) + 5 + msgp.ArrayHeaderSize
	for zrsw := range z.Tags {
		if z.Tags[zrsw] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zrsw].Msgsize()
		}
	}
	s += 7 + msgp.StringPrefixSize + len(z.Format)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *IndexQuery) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Query":
			z.Query, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Value":
			z.Value, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Page":
			z.Page, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Tags":
			var zpez uint32
			zpez, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zpez) {
				z.Tags = (z.Tags)[:zpez]
			} else {
				z.Tags = make([]*repr.Tag, zpez)
			}
			for zkgt := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zkgt] = nil
				} else {
					if z.Tags[zkgt] == nil {
						z.Tags[zkgt] = new(repr.Tag)
					}
					err = z.Tags[zkgt].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "HasData":
			z.HasData, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "InCache":
			z.InCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "Format":
			z.Format, err = dc.ReadString()
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
func (z *IndexQuery) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "Query"
	err = en.Append(0x87, 0xa5, 0x51, 0x75, 0x65, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Query)
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
	// write "Page"
	err = en.Append(0xa4, 0x50, 0x61, 0x67, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Page)
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
	for zkgt := range z.Tags {
		if z.Tags[zkgt] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zkgt].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "HasData"
	err = en.Append(0xa7, 0x48, 0x61, 0x73, 0x44, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.HasData)
	if err != nil {
		return
	}
	// write "InCache"
	err = en.Append(0xa7, 0x49, 0x6e, 0x43, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.InCache)
	if err != nil {
		return
	}
	// write "Format"
	err = en.Append(0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Format)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *IndexQuery) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "Query"
	o = append(o, 0x87, 0xa5, 0x51, 0x75, 0x65, 0x72, 0x79)
	o = msgp.AppendString(o, z.Query)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendString(o, z.Value)
	// string "Page"
	o = append(o, 0xa4, 0x50, 0x61, 0x67, 0x65)
	o = msgp.AppendUint32(o, z.Page)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zkgt := range z.Tags {
		if z.Tags[zkgt] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zkgt].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "HasData"
	o = append(o, 0xa7, 0x48, 0x61, 0x73, 0x44, 0x61, 0x74, 0x61)
	o = msgp.AppendBool(o, z.HasData)
	// string "InCache"
	o = append(o, 0xa7, 0x49, 0x6e, 0x43, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.InCache)
	// string "Format"
	o = append(o, 0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	o = msgp.AppendString(o, z.Format)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *IndexQuery) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Query":
			z.Query, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Page":
			z.Page, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zqyh uint32
			zqyh, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zqyh) {
				z.Tags = (z.Tags)[:zqyh]
			} else {
				z.Tags = make([]*repr.Tag, zqyh)
			}
			for zkgt := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zkgt] = nil
				} else {
					if z.Tags[zkgt] == nil {
						z.Tags[zkgt] = new(repr.Tag)
					}
					bts, err = z.Tags[zkgt].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "HasData":
			z.HasData, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "InCache":
			z.InCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "Format":
			z.Format, bts, err = msgp.ReadStringBytes(bts)
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
func (z *IndexQuery) Msgsize() (s int) {
	s = 1 + 6 + msgp.StringPrefixSize + len(z.Query) + 6 + msgp.StringPrefixSize + len(z.Value) + 5 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zkgt := range z.Tags {
		if z.Tags[zkgt] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zkgt].Msgsize()
		}
	}
	s += 8 + msgp.BoolSize + 8 + msgp.BoolSize + 7 + msgp.StringPrefixSize + len(z.Format)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricQuery) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Start":
			z.Start, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "End":
			z.End, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Step":
			z.Step, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Agg":
			z.Agg, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "MaxPoints":
			z.MaxPoints, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Tags":
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
			for zyzr := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zyzr] = nil
				} else {
					if z.Tags[zyzr] == nil {
						z.Tags[zyzr] = new(repr.Tag)
					}
					err = z.Tags[zyzr].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "InCache":
			z.InCache, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "Format":
			z.Format, err = dc.ReadString()
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
func (z *MetricQuery) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "Target"
	err = en.Append(0x89, 0xa6, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "Start"
	err = en.Append(0xa5, 0x53, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Start)
	if err != nil {
		return
	}
	// write "End"
	err = en.Append(0xa3, 0x45, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.End)
	if err != nil {
		return
	}
	// write "Step"
	err = en.Append(0xa4, 0x53, 0x74, 0x65, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Step)
	if err != nil {
		return
	}
	// write "Agg"
	err = en.Append(0xa3, 0x41, 0x67, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Agg)
	if err != nil {
		return
	}
	// write "MaxPoints"
	err = en.Append(0xa9, 0x4d, 0x61, 0x78, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.MaxPoints)
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
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zyzr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "InCache"
	err = en.Append(0xa7, 0x49, 0x6e, 0x43, 0x61, 0x63, 0x68, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.InCache)
	if err != nil {
		return
	}
	// write "Format"
	err = en.Append(0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Format)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricQuery) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "Target"
	o = append(o, 0x89, 0xa6, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "Start"
	o = append(o, 0xa5, 0x53, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendInt64(o, z.Start)
	// string "End"
	o = append(o, 0xa3, 0x45, 0x6e, 0x64)
	o = msgp.AppendInt64(o, z.End)
	// string "Step"
	o = append(o, 0xa4, 0x53, 0x74, 0x65, 0x70)
	o = msgp.AppendUint32(o, z.Step)
	// string "Agg"
	o = append(o, 0xa3, 0x41, 0x67, 0x67)
	o = msgp.AppendUint32(o, z.Agg)
	// string "MaxPoints"
	o = append(o, 0xa9, 0x4d, 0x61, 0x78, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendUint32(o, z.MaxPoints)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zyzr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "InCache"
	o = append(o, 0xa7, 0x49, 0x6e, 0x43, 0x61, 0x63, 0x68, 0x65)
	o = msgp.AppendBool(o, z.InCache)
	// string "Format"
	o = append(o, 0xa6, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74)
	o = msgp.AppendString(o, z.Format)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricQuery) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zzpf uint32
	zzpf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zzpf > 0 {
		zzpf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Start":
			z.Start, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "End":
			z.End, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Step":
			z.Step, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Agg":
			z.Agg, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "MaxPoints":
			z.MaxPoints, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zrfe uint32
			zrfe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrfe) {
				z.Tags = (z.Tags)[:zrfe]
			} else {
				z.Tags = make([]*repr.Tag, zrfe)
			}
			for zyzr := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zyzr] = nil
				} else {
					if z.Tags[zyzr] == nil {
						z.Tags[zyzr] = new(repr.Tag)
					}
					bts, err = z.Tags[zyzr].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "InCache":
			z.InCache, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "Format":
			z.Format, bts, err = msgp.ReadStringBytes(bts)
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
func (z *MetricQuery) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 6 + msgp.Int64Size + 4 + msgp.Int64Size + 5 + msgp.Uint32Size + 4 + msgp.Uint32Size + 10 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zyzr].Msgsize()
		}
	}
	s += 8 + msgp.BoolSize + 7 + msgp.StringPrefixSize + len(z.Format)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Resolution) DecodeMsg(dc *msgp.Reader) (err error) {
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
func (z Resolution) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Resolution"
	err = en.Append(0x82, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
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
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Resolution) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Resolution"
	o = append(o, 0x82, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Resolution) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
func (z Resolution) Msgsize() (s int) {
	s = 1 + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size
	return
}

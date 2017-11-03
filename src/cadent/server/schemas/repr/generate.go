//go:generate protoc --proto_path=../../../../cadent/vendor/:./ --gogofaster_out=. repr.proto
//go:generate msgp -tests -o repr_msgp.go --file repr.pb.go
//go:generate msgp -tests -o tags_msgp.go --file tags.go
//go:generate msgp -tests -o repr_list_msgp.go --file repr.go

// XXX SEE NOTE BELOW AFTER THIS STEP

/* NOTE after this edit repr.pb.go to put

after generation you will need to replace []*Tag w/ SortingTags in the repr.pb.go

type StatName struct {
	Key             string      `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	XXX_uniqueId    uint64      `protobuf:"varint,2,opt,name=uniqueId,proto3" json:"-" msg:"-"`
	XXX_uniqueIdstr string      `protobuf:"bytes,3,opt,name=uniqueIdstr,proto3" json:"-"  msg:"-"`
	Resolution      uint32      `protobuf:"varint,4,opt,name=resolution,proto3" json:"resolution,omitempty"`
	Ttl             uint32      `protobuf:"varint,5,opt,name=ttl,proto3" json:"ttl,omitempty"`
	TagMode         TagMode     `protobuf:"varint,6,opt,name=tagMode,proto3,enum=repr.TagMode" json:"tagMode,omitempty"`
	HashMode        HashMode    `protobuf:"varint,7,opt,name=hashMode,proto3,enum=repr.HashMode" json:"hashMode,omitempty"`
	Tags            SortingTags `protobuf:"bytes,13,rep,name=tags" json:"tags,omitempty"`
	MetaTags        SortingTags `protobuf:"bytes,14,rep,name=meta_tags,json=metaTags" json:"meta_tags,omitempty"`
}

and

THEN run

go:generate easyjson -all repr.pb.go
go:generate easyjson -all tags.go
go:generate easyjson -all repr.go


After the easyjson runs on repr.pb.go you need to EDIT the to include the CheckFloat for NANs


func easyjsonCee91ed5EncodeCadentServerSchemasRepr(out *jwriter.Writer, in StatRepr) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Name != nil {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"name\":")
		if in.Name == nil {
			out.RawString("null")
		} else {
			out.Raw((*in.Name).MarshalJSON())
		}
	}
	if in.Time != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"time\":")
		out.Int64(int64(in.Time))
	}
	if in.Min != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"min\":")
		out.Float64(CheckFloat(float64(in.Min)))
	}
	if in.Max != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"max\":")
		out.Float64(CheckFloat(float64(in.Max)))
	}
	if in.Last != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"last\":")
		out.Float64(CheckFloat(float64(in.Last)))
	}
	if in.Sum != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"sum\":")
		out.Float64(CheckFloat(float64(in.Sum)))
	}
	if in.Count != 0 {
		if !first {
			out.RawByte(',')
		}
		first = false
		out.RawString("\"count\":")
		out.Int64(int64(in.Count))
	}
	out.RawByte('}')
}

*/

package repr

// AUTOGENERATED FILE: easyjson marshaller/unmarshallers.

package repr

import (
	json "encoding/json"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ = json.RawMessage{}
	_ = jlexer.Lexer{}
	_ = jwriter.Writer{}
)

func easyjson1d20e661DecodeCadentServerSchemasRepr(in *jlexer.Lexer, out *ReprList) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "min_time":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.MinTime).UnmarshalJSON(data))
			}
		case "max_time":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.MaxTime).UnmarshalJSON(data))
			}
		case "stats":
			if in.IsNull() {
				in.Skip()
				out.Reprs = nil
			} else {
				in.Delim('[')
				if !in.IsDelim(']') {
					out.Reprs = make([]StatRepr, 0, 1)
				} else {
					out.Reprs = []StatRepr{}
				}
				for !in.IsDelim(']') {
					var v1 StatRepr
					if data := in.Raw(); in.Ok() {
						in.AddError((v1).UnmarshalJSON(data))
					}
					out.Reprs = append(out.Reprs, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson1d20e661EncodeCadentServerSchemasRepr(out *jwriter.Writer, in ReprList) {
	out.RawByte('{')
	first := true
	_ = first
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"min_time\":")
	out.Raw((in.MinTime).MarshalJSON())
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"max_time\":")
	out.Raw((in.MaxTime).MarshalJSON())
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"stats\":")
	if in.Reprs == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
		out.RawString("null")
	} else {
		out.RawByte('[')
		for v2, v3 := range in.Reprs {
			if v2 > 0 {
				out.RawByte(',')
			}
			out.Raw((v3).MarshalJSON())
		}
		out.RawByte(']')
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v ReprList) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson1d20e661EncodeCadentServerSchemasRepr(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v ReprList) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson1d20e661EncodeCadentServerSchemasRepr(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *ReprList) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson1d20e661DecodeCadentServerSchemasRepr(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *ReprList) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson1d20e661DecodeCadentServerSchemasRepr(l, v)
}

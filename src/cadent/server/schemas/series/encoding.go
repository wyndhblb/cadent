//go:generate protoc --proto_path=../../../../cadent/vendor/:./:../../../../ --gogofaster_out=. series.proto
//go:generate msgp -o series_msgp.go --file series.pb.go
//go:generate easyjson -all series.pb.go

package series

type SendEncoding uint8

const (
	ENCODE_JSON SendEncoding = iota
	ENCODE_MSGP
	ENCODE_PROTOBUF
)

func SendEncodingFromString(enc string) SendEncoding {
	switch enc {
	case "json":
		return ENCODE_JSON
	case "protobuf":
		return ENCODE_PROTOBUF
	default:
		return ENCODE_MSGP
	}
}

type MessageType uint8

const (
	MSG_SERIES MessageType = iota
	MSG_SINGLE
	MSG_UNPROCESSED
	MSG_UID
	MSG_RAW
	MSG_ANY
	MSG_WRITTEN
	MSG_SINGLE_LIST
	MSG_UNPROCESSED_LIST
	MSG_RAW_LIST
	MSG_ANY_LIST
)

func MetricTypeFromString(enc string) MessageType {
	switch enc {
	case "series":
		return MSG_SERIES
	case "unprocessed":
		return MSG_UNPROCESSED
	case "uid":
		return MSG_UID
	case "raw":
		return MSG_RAW
	case "single":
		return MSG_SINGLE
	case "any", "all":
		return MSG_ANY
	case "written":
		return MSG_WRITTEN
	case "single_list":
		return MSG_SINGLE_LIST
	case "unprocessed_list":
		return MSG_UNPROCESSED_LIST
	case "raw_list":
		return MSG_RAW_LIST
	case "all_list", "any_list":
		return MSG_ANY_LIST
	default:
		return MSG_SINGLE
	}
}

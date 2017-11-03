package series

import (
	"cadent/server/schemas/repr"
	"testing"
	"time"
)

func getTestStat() *repr.StatRepr {

	t := time.Date(2009, 10, 23, 5, 54, 9, 0, time.UTC)
	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:        "moo.goo.org",
			Tags:       *repr.SortingTagsFromString("loo=goo,monkey=house"),
			Ttl:        1000,
			Resolution: 10,
		},
		Time:  t.Unix(),
		Min:   1.0,
		Max:   100.0,
		Count: 12,
		Sum:   400.0,
		Last:  5.0,
	}
}

func Test_KRawMetric(t *testing.T) {

	s := getTestStat()

	v := KRawMetric{
		RawMetric: RawMetric{
			Metric: s.Name.Key,
			Time:   s.Time,
			Value:  s.Sum,
		},
	}

	gr := v.Repr()
	if gr.Sum != s.Sum {
		t.Fatal("Invalid sum")
	}
	if gr.Count != 1 {
		t.Fatal("Invalid count")
	}
	if gr.Name.Key != s.Name.Key {
		t.Fatal("Invalid name")
	}
	if gr.Time != s.Time {
		t.Fatal("Invalid name")
	}

	if v.Id() != s.Name.Key {
		t.Fatal("Invalid id")
	}

	for _, enc := range []SendEncoding{ENCODE_JSON, ENCODE_MSGP, ENCODE_PROTOBUF} {

		v.SetSendEncoding(enc)
		j, err := v.Encode()
		if err != nil {
			t.Fatal(err)
		}

		err = v.Decode(j)
		if err != nil {
			t.Fatal("failed decode")
		}

		if v.Metric != s.Name.Key {
			t.Fatal("Invalid decode name")
		}
	}

}

func Test_KUnProcessedMetric(t *testing.T) {

	s := getTestStat()

	v := KUnProcessedMetric{
		UnProcessedMetric: UnProcessedMetric{
			Metric: s.Name.Key,
			Tags:   s.Name.Tags,
			Time:   s.Time,
			Sum:    s.Sum,
			Count:  s.Count,
			Min:    s.Min,
			Max:    s.Max,
			Last:   s.Last,
		},
	}

	gr := v.Repr()
	if gr.Sum != s.Sum {
		t.Fatal("Invalid sum")
	}
	if gr.Max != s.Max {
		t.Fatal("Invalid sum")
	}
	if gr.Count != s.Count {
		t.Fatal("Invalid count")
	}
	if gr.Name.Key != s.Name.Key {
		t.Fatal("Invalid name")
	}
	if gr.Time != s.Time {
		t.Fatal("Invalid name")
	}

	if v.Id() != s.Name.Key {
		t.Fatal("Invalid id")
	}

	for _, enc := range []SendEncoding{ENCODE_JSON, ENCODE_MSGP, ENCODE_PROTOBUF} {

		v.SetSendEncoding(enc)
		j, err := v.Encode()
		if err != nil {
			t.Fatal(err)
		}

		err = v.Decode(j)
		if err != nil {
			t.Fatal("failed decode")
		}

		if v.Metric != s.Name.Key {
			t.Fatal("Invalid decode name")
		}
	}

}

func Test_KSingleMetric(t *testing.T) {

	s := getTestStat()

	v := KSingleMetric{
		SingleMetric: SingleMetric{
			Id:     uint64(s.Name.UniqueId()),
			Uid:    s.Name.UniqueIdString(),
			Tags:   s.Name.Tags,
			Metric: s.Name.Key,
			Time:   s.Time,
			Sum:    s.Sum,
			Count:  s.Count,
			Min:    s.Min,
			Max:    s.Max,
			Last:   s.Last,
		},
	}

	gr := v.Repr()
	if gr.Sum != s.Sum {
		t.Fatal("Invalid sum")
	}
	if gr.Name.UniqueId() != s.Name.UniqueId() {
		t.Fatalf("Invalid id %d, %d", gr.Name.UniqueId(), s.Name.UniqueId())
	}
	if gr.Name.UniqueIdString() != s.Name.UniqueIdString() {
		t.Fatal("Invalid id string")
	}
	if gr.Max != s.Max {
		t.Fatal("Invalid sum")
	}
	if gr.Count != s.Count {
		t.Fatal("Invalid count")
	}
	if gr.Name.Key != s.Name.Key {
		t.Fatal("Invalid name")
	}
	if gr.Time != s.Time {
		t.Fatal("Invalid name")
	}

	if v.Id() != s.Name.UniqueIdString() {
		t.Fatal("Invalid id")
	}

	for _, enc := range []SendEncoding{ENCODE_JSON, ENCODE_MSGP, ENCODE_PROTOBUF} {

		v.SetSendEncoding(enc)
		j, err := v.Encode()
		if err != nil {
			t.Fatal(err)
		}

		err = v.Decode(j)
		if err != nil {
			t.Fatal("failed decode")
		}

		if v.Metric != s.Name.Key {
			t.Fatal("Invalid decode name")
		}
	}
}

func Test_KMetric(t *testing.T) {

	s := getTestStat()

	v := KMetric{
		AnyMetric: AnyMetric{
			Single: &SingleMetric{
				Id:     uint64(s.Name.UniqueId()),
				Uid:    s.Name.UniqueIdString(),
				Tags:   s.Name.Tags,
				Metric: s.Name.Key,
				Time:   s.Time,
				Sum:    s.Sum,
				Count:  s.Count,
				Min:    s.Min,
				Max:    s.Max,
				Last:   s.Last,
			},
		},
	}

	gr := v.Repr()
	if gr.Sum != s.Sum {
		t.Fatal("Invalid sum")
	}
	if gr.Name.UniqueId() != s.Name.UniqueId() {
		t.Fatalf("Invalid id %d, %d", gr.Name.UniqueId(), s.Name.UniqueId())
	}
	if gr.Name.UniqueIdString() != s.Name.UniqueIdString() {
		t.Fatal("Invalid id string")
	}
	if gr.Max != s.Max {
		t.Fatal("Invalid max")
	}
	if gr.Min != s.Min {
		t.Fatal("Invalid min")
	}
	if gr.Count != s.Count {
		t.Fatal("Invalid count")
	}
	if gr.Name.Key != s.Name.Key {
		t.Fatal("Invalid name")
	}
	if gr.Time != s.Time {
		t.Fatal("Invalid name")
	}

	if v.Id() != s.Name.UniqueIdString() {
		t.Fatal("Invalid id")
	}
	for _, enc := range []SendEncoding{ENCODE_JSON, ENCODE_MSGP, ENCODE_PROTOBUF} {

		v.SetSendEncoding(enc)
		j, err := v.Encode()
		if err != nil {
			t.Fatal(err)
		}

		err = v.Decode(j)
		if err != nil {
			t.Fatal("failed decode")
		}

		if v.Single.Metric != s.Name.Key {
			t.Fatal("Invalid decode name")
		}
	}
}

func Test_KMetricList(t *testing.T) {

	s := getTestStat()

	sing := &SingleMetric{
		Id:     uint64(s.Name.UniqueId()),
		Uid:    s.Name.UniqueIdString(),
		Tags:   s.Name.Tags,
		Metric: s.Name.Key,
		Time:   s.Time,
		Sum:    s.Sum,
		Count:  s.Count,
		Min:    s.Min,
		Max:    s.Max,
		Last:   s.Last,
	}

	unpr := &UnProcessedMetric{
		Tags:   s.Name.Tags,
		Metric: s.Name.Key,
		Time:   s.Time,
		Sum:    s.Sum,
		Count:  s.Count,
		Min:    s.Min,
		Max:    s.Max,
		Last:   s.Last,
	}

	rawm := &RawMetric{
		Tags:   s.Name.Tags,
		Metric: s.Name.Key,
		Time:   s.Time,
		Value:  s.Sum,
	}

	meList := AnyMetricList{Single: []*SingleMetric{sing, sing, sing}}
	unprList := AnyMetricList{Unprocessed: []*UnProcessedMetric{unpr, unpr, unpr}}
	rawList := AnyMetricList{Raw: []*RawMetric{rawm, rawm, rawm}}

	anyList := []AnyMetricList{
		meList,
		unprList,
		rawList,
	}

	for _, onlist := range anyList {
		v := KMetricList{
			AnyMetricList: onlist,
		}

		reprs := v.Reprs()
		for _, gr := range reprs {

			if gr.Sum != s.Sum {
				t.Fatal("Invalid sum")
			}
			if gr.Name.UniqueId() != s.Name.UniqueId() {
				t.Fatalf("Invalid id %d, %d", gr.Name.UniqueId(), s.Name.UniqueId())
			}
			if gr.Name.UniqueIdString() != s.Name.UniqueIdString() {
				t.Fatal("Invalid id string")
			}
			if onlist.Raw == nil {
				if gr.Max != s.Max {
					t.Fatal("Invalid max")
				}
				if gr.Min != s.Min {
					t.Fatal("Invalid min")
				}
				if gr.Count != s.Count {
					t.Fatal("Invalid count")
				}
			}

			if gr.Name.Key != s.Name.Key {
				t.Fatal("Invalid name")
			}
			if gr.Time != s.Time {
				t.Fatal("Invalid name")
			}
		}

		ids := v.Ids()
		if onlist.Unprocessed != nil || onlist.Raw != nil {
			for _, on := range ids {
				if on != s.Name.Key {
					t.Fatal("Invalid id")
				}
			}
		}
		if onlist.Single != nil {
			for _, on := range ids {
				if on != s.Name.UniqueIdString() {
					t.Fatal("Invalid id")
				}
			}
		}

		for _, enc := range []SendEncoding{ENCODE_JSON, ENCODE_MSGP, ENCODE_PROTOBUF} {

			v.SetSendEncoding(enc)
			j, err := v.Encode()
			if err != nil {
				t.Fatal(err)
			}

			err = v.Decode(j)
			if err != nil {
				t.Fatal("failed decode")
			}

			if onlist.Unprocessed != nil {

				if v.Unprocessed[0].Metric != s.Name.Key {
					t.Fatal("Invalid decode name")
				}
			}
			if onlist.Single != nil {
				if v.Single[0].Metric != s.Name.Key {
					t.Fatal("Invalid decode name")
				}
			}
			if onlist.Raw != nil {
				if v.Raw[0].Metric != s.Name.Key {
					t.Fatal("Invalid decode name")
				}
			}
		}

	}
}

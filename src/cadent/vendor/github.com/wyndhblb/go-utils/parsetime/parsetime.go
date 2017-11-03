/*
Parse random strings into times and durations
*/

package parsetime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrorCouldNotParseNumberToTime = errors.New("Could not parse the number into a time")

// ParseTime .. given a string find a proper unix timestamp
// given an arbitrary string try to get a time object
// strings can be of the form and are tried in this ordering
//
// 2006-01-02T15:04:05Z07:00 (RFC3339)
// 2006-01-02 15:04:05
// 2006-01-02
// 2006-01-02 15:04
// 2006-01-02 15
// Mon Jan 2 15:04:05 -0700 MST 2006
//
// now
// today
// yesterday
// lastweek
//
// and "durations from the epoch"
// 1s, 1sec
// 1m, 1min
// 1d, 1day
// 1mon, 1month
// 1y, 1year
//
// finally attempt to parse an epoch time
//
// 1485984333
// 1485984333.123
// 1485984333.123123
// 1485984333.123123123
//
func ParseTime(st string) (time.Time, error) {
	st = strings.TrimSpace(st)

	// first see if it's a "date string"
	// of the form 2015-07-01T20:10:30.781Z
	_time, err := time.Parse("2006-01-02T15:04:05Z07:00", st)
	if err == nil {
		return _time, nil
	}

	_time, err = time.Parse("2006-01-02 15:04:05", st)
	if err == nil {
		return _time, nil
	}

	_time, err = time.Parse("2006-01-02 15:04:05", st+" 00:00:00")
	if err == nil {
		return _time, nil
	}

	_time, err = time.Parse("2006-01-02 15:04:05", st+":00:00")
	if err == nil {
		return _time, nil
	}

	_time, err = time.Parse("2006-01-02 15:04:05", st+":00")
	if err == nil {
		return _time, nil
	}

	// or Mon Jan 2 15:04:05 -0700 MST 2006
	_time, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", st)
	if err == nil {
		return _time, nil
	}

	st = strings.ToLower(st)
	if st == "now" || st == "today" {
		return time.Now(), nil
	}

	unixT := time.Now().Unix()
	if st == "yesterday" {
		st = "-1d"
	}
	if st == "lastweek" {
		st = "-7d"
	}
	if strings.HasSuffix(st, "s") || strings.HasSuffix(st, "sec") || strings.HasSuffix(st, "second") {
		items := strings.Split(st, "s")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(unixT+i, 0), nil
	}

	if strings.HasSuffix(st, "m") || strings.HasSuffix(st, "min") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(unixT+i*60, 0), nil
	}

	if strings.HasSuffix(st, "h") || strings.HasSuffix(st, "hour") {
		items := strings.Split(st, "h")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(unixT+i*60*60, 0), nil
	}

	if strings.HasSuffix(st, "d") || strings.HasSuffix(st, "day") {
		items := strings.Split(st, "d")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Now().AddDate(0, 0, int(i))
		return t, nil
	}
	if strings.HasSuffix(st, "mon") || strings.HasSuffix(st, "month") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Now().AddDate(0, int(i), 0)
		return t, nil
	}
	if strings.HasSuffix(st, "y") || strings.HasSuffix(st, "year") {
		items := strings.Split(st, "y")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Now().AddDate(int(i), 0, 0)
		return t, nil
	}

	// if it's an int already, we're good
	if strings.Contains(st, ".") {
		spl := strings.Split(st, ".")
		if len(spl) > 2 {
			return time.Time{}, ErrorCouldNotParseNumberToTime
		}
		i, err := strconv.ParseInt(spl[0], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		if len(spl) == 2 {
			// milli
			nanos := int64(0)
			switch len(spl[1]) {
			case 3:
				n, err := strconv.ParseInt(spl[1], 10, 64)
				if err != nil {
					return time.Time{}, err
				}
				nanos = int64(n) * int64(time.Millisecond)
			case 6:
				n, err := strconv.ParseInt(spl[1], 10, 64)
				if err != nil {
					return time.Time{}, err
				}
				nanos = int64(n) * int64(time.Microsecond)
			case 9:
				n, err := strconv.ParseInt(spl[1], 10, 64)
				if err != nil {
					return time.Time{}, err
				}
				nanos = int64(n)
			default:
				return time.Time{}, ErrorCouldNotParseNumberToTime
			}
			return time.Unix(i, nanos), nil
		}
		return time.Unix(int64(i), 0), nil

	}
	i, err := strconv.ParseInt(st, 10, 64)
	if err == nil {
		return time.Unix(i, 0), nil
	}

	return time.Time{}, fmt.Errorf("Time `%s` could not be parsed :: %v", st, err)

}

// ParseDuration .. given a string find a proper time.Duration
// strings can be of the form
// now -> nil
// 1s, 1sec, 1second
// 1m, 1min
// 1h, 1hour
// 1d, 1day
// 1mon, 1month
// 1y, 1year
func ParseDuration(st string) (time.Duration, error) {
	st = strings.TrimSpace(strings.ToLower(st))
	nilDur := time.Duration(0)
	iSec := int64(time.Second)

	if st == "now" || st == "today" {
		return nilDur, nil
	}

	if strings.HasSuffix(st, "s") || strings.HasSuffix(st, "sec") || strings.HasSuffix(st, "second") {
		items := strings.Split(st, "s")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(i * iSec), nil
	}

	if strings.HasSuffix(st, "m") || strings.HasSuffix(st, "min") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(iSec * i * 60), nil
	}

	if strings.HasSuffix(st, "h") || strings.HasSuffix(st, "hour") {
		items := strings.Split(st, "h")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(iSec * i * 60 * 60), nil
	}

	if strings.HasSuffix(st, "d") || strings.HasSuffix(st, "day") {
		items := strings.Split(st, "d")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(iSec * i * 60 * 60 * 24), nil
	}
	if strings.HasSuffix(st, "mon") || strings.HasSuffix(st, "month") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(iSec * i * 60 * 60 * 24 * 30), nil
	}
	if strings.HasSuffix(st, "y") || strings.HasSuffix(st, "year") {
		items := strings.Split(st, "y")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return nilDur, err
		}
		return time.Duration(iSec * i * 60 * 60 * 24 * 365), nil
	}

	return time.Duration(0), fmt.Errorf("Duration `%s` could not be parsed", st)

}

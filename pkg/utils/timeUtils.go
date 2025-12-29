package utils

import (
	"AI_class/def"
	"errors"
	"time"
)

var DateChangeError = errors.New("时间转换字符串错误")
var StringChangeDateError = errors.New("字符串转换时间错误")

func ParseStringToUnix(timeStr string) (int64, error) {
	if timeStr == "" {
		return 0, StringChangeDateError
	}

	var timeUnix time.Time
	var err error
	switch len(timeStr) {
	case 19:
		timeUnix, err = time.ParseInLocation(def.TimeFormat, timeStr, time.Local)
	case 10:
		timeUnix, err = time.ParseInLocation(def.TimeShortFormat, timeStr, time.Local)
	}

	if err != nil {
		return 0, StringChangeDateError
	}

	return timeUnix.Unix(), nil
}

// 依据指定的格式进行转化，转化时间戳方式进行转换
func ParseUnixToString(timeUnix int64, chanegWay int, timeFormat int) (string, error) {
	if timeUnix == 0 {
		return "", DateChangeError
	}

	var timeUnixTime time.Time
	//进行时间转换秒级时间戳
	switch chanegWay {
	case 1: //秒级
		timeUnixTime = time.Unix(timeUnix, 0)
	case 2:
		timeUnixTime = time.Unix(timeUnix, timeUnix*1000000)
	case 3:
		timeUnixTime = time.UnixMicro(timeUnix)
	}

	if timeUnixTime.IsZero() {
		return "", StringChangeDateError
	}

	var timeFormatStr string
	if timeFormat == 1 {
		timeFormatStr = def.TimeFormat
	} else {
		timeFormatStr = def.TimeShortFormat
	}

	return timeUnixTime.Format(timeFormatStr), nil

}

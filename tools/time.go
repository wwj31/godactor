package tools

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	SecondsPerDay       = 24 * 60 * 60
	Day                 = 24 * time.Hour
	StdTimeLayout       = "2006-01-02 15:04:05.000"
	StdTimeSimpleLayout = "2006-1-2 15:4:5"
	StdDateFormat       = "2006-01-02"
)

var (
	TimeOffset     int64
	TimeOffsetPath = "./.timeoffset"
)

func init() {
	bytes, err := os.ReadFile(TimeOffsetPath)
	if err != nil {
		return
	}
	str := string(bytes)
	t, err := strconv.Atoi(str)
	if err != nil {
		return
	}
	TimeOffset = int64(t)
}

func ModifyTimeOffset(add int64) {
	TimeOffset += add
	file, err := os.Create(TimeOffsetPath)
	if err != nil {
		panic(err)
	}
	str := strconv.Itoa(int(TimeOffset))
	_, err = file.Write([]byte(str))
	if err != nil {
		panic(err)
	}
}

func Now() time.Time {
	return time.Now().UTC().Add(time.Duration(TimeOffset))
}

func TimeFormat(data time.Time) string {
	//time.RFC3339Nano
	return data.Format(StdTimeLayout)
}

func TimeParse(data string) (t time.Time, err error) {
	t, err = time.Parse(StdTimeLayout, data)
	if err != nil {
		return
	}
	return
}

// NextIntervalTime 以当天开始时间为初始值 间隔 intervalSeconds触发一次，返回下次触发的时间
func NextIntervalTime(t time.Time, intervalSeconds int) time.Time {
	if intervalSeconds <= 0 {
		fmt.Println("wrong1 intervalSeconds")
		return Now()
	}
	if SecondsPerDay%intervalSeconds != 0 {
		fmt.Println("wrong2 intervalSeconds")
	}

	beginOfDay := NewTimeEx(t).BeginOfToday()
	duration := int(t.Sub(beginOfDay).Seconds())
	next := intervalSeconds * (duration/intervalSeconds + 1)

	return beginOfDay.Add(time.Second * time.Duration(next))
}

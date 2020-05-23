package log

import (
    "fmt"
    "time"
    "github.com/hjjg200/go-together"
)

var Debug = false
var DebugFilter = &Filter{}

const (
    debugPrefixInfo = "INFO"
    debugPrefixTime = "TIME"
)

func(lgr *Logger) debug(category, prefix string, args ...interface{}) {

    args = append(
        []interface{}{Magenta(prefix + ": " + category)},
        args...,
    )

    switch prefix {
    case debugPrefixInfo:
        args = append(args, lgr.callers(4)...)
    }

    lgr.println(prefixDebug, args...)

}

func(lgr *Logger) Debugln(category string, args ...interface{}) {

    if !(Debug && DebugFilter.Filter(category)) {
        return
    }
    lgr.debug(category, debugPrefixInfo, args...)

}

// TIMER ---

type Timer struct {
    lgr      *Logger
    last     time.Time
    category string
}

var timeLockers = together.NewLockerRoom()
var timeCategories = make(map[string] time.Duration)
func(lgr *Logger) Timer(category string) *Timer {
    return &Timer{
        lgr: lgr,
        last: time.Now(),
        category: category,
    }
}

func(tm *Timer) Stop() {

    if !(Debug && DebugFilter.Filter(tm.category)) {
        return
    }

    past := time.Now().Sub(tm.last)

    timeLockers.Lock(tm.category)
    d, _  := timeCategories[tm.category]
    total := d + past
    timeCategories[tm.category] = total
    timeLockers.Unlock(tm.category)

    tm.lgr.debug(
        tm.category, debugPrefixTime,
        fmt.Sprintf("+%v", past), fmt.Sprintf("total: %v", total),
    )

}
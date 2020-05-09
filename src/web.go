package main

// CONFIG ---

type WebConfig struct {
    Durations       []int  `json:"durations"`
    FormatValue     string `json:"format.value"`
    FormatYAxis     string `json:"format.yAxis"`
    FormatDateLong  string `json:"format.date.long"`
    FormatDateShort string `json:"format.date.short"`
}

var DefaultWebConfig = WebConfig{
    Durations:       []int{60*3, 60*24, 60*24*5},
    FormatValue:     "{.2f}",
    FormatYAxis:     "{}",
    FormatDateLong:  "Y-MM-DD[T]HH:mm:ssZ", // ISO 8601
    FormatDateShort: "MMM DD HH:mm",
}
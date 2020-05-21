package monitor

import (
    "strconv"
    "strings"
    "time"
)

var systemStartTime time.Time
func init() {
    ut, err := GetUptime()
    if err != nil {
        panic(err)
    }

    systemStartTime = time.Now().Add(
        time.Duration(float64(time.Second) * ut * -1.0),
    )
}

func GetUptime() (float64, error) {
    
    cat, err := readFile("/proc/uptime")
    if err != nil {
        return 0.0, err
    }

    splits := strings.Split(cat, " ")
    return strconv.ParseFloat(splits[0], 64)

}
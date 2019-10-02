package monitor

import (
    "fmt"
)

// GetLoadAverage returns the load average
// 1m, 5m, 15m in order

func GetLoadAverage() (map[string] float64, error) {

    cat, err := readFile("/proc/loadavg")
    if err != nil {
        return nil, err
    }

    // Parse
    var (
        la1m float64
        la5m float64
        la15m float64
   )
    n, err := fmt.Sscanf(cat, "%f %f %f", &la1m, &la5m, &la15m)
    if n != 3 || err != nil {
        return nil, fmt.Errorf("Failed to parse /proc/loadavg")
    }

    // Return
    return map[string] float64 {
        "1m": la1m, "5m": la5m, "15m": la15m,
    }, nil

}

func GetLoadAveragePerCpu() (map[string] float64, error) {
    m, err := GetLoadAverage()
    if err != nil {
        return nil, err
    }
    cc := GetCpuCount()
    for k, v := range m {
        m[k] = v / cc
    }
    return m, nil
}
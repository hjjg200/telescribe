package monitor

import (
    "fmt"
    "testing"
    "time"
)

func TestProcessEntireMemoryUsage(t *testing.T) {

    // Test if the sum of memory usage of entire processes is equal to system memory usage
    
    parseProcesses()

    for {

        time.Sleep(1 * time.Second)

        parseProcesses()
        processMutex.RLock()

        total := uint64(0)

        for _, ppidmp := range parsedPidSmaps {
            total += ppidmp.uss
        }
        processMutex.RUnlock()

        sys, _ := GetMemoryUsage()

        fmt.Println(
            "Entire Processes:", float64(total) / GetMemoryTotal() * 100.0,
            "System Memory:", sys,
        )

    }

}
package monitor

import (
    "fmt"
    "testing"
    "time"
)


func TestDiskIOTicks(t *testing.T) {

    // Test if io ticks = read ticks + write ticks
    parseDiskStats()
    prev := parsedDiskStats
    for {
        time.Sleep(1 * time.Second)

        parseDiskStats()
        pds := parsedDiskStats

        for name := range pds {
            p0 := prev[name]
            p1 := pds[name]
            dr := p1.readTicks - p0.readTicks
            dw := p1.writeTicks - p0.writeTicks
            di := p1.ioTicks - p0.ioTicks

            fmt.Println(
                name + ": (rticks + wticks) / ioticks =",
                float64(dr + dw) / float64(di),
            )
        }

        prev = pds

    }

// Result: read ticks + write ticks != io ticks

// xvda1: (rticks + wticks) / ioticks = 1.4861111111111112
// xvda1: (rticks + wticks) / ioticks = 1.45
// xvda1: (rticks + wticks) / ioticks = NaN
// xvda1: (rticks + wticks) / ioticks = 1.0277777777777777
// xvda1: (rticks + wticks) / ioticks = NaN
// xvda1: (rticks + wticks) / ioticks = 1.0178571428571428
// xvda1: (rticks + wticks) / ioticks = 0.82
// xvda1: (rticks + wticks) / ioticks = 1.4107142857142858
// xvda1: (rticks + wticks) / ioticks = NaN
// xvda1: (rticks + wticks) / ioticks = 0.875
// xvda1: (rticks + wticks) / ioticks = NaN

}
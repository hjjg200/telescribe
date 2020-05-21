package monitor

// #include <unistd.h>
import "C"

import (
    "time"
)

var (
    c_SC_CLK_TCK = C.sysconf(C._SC_CLK_TCK)
)

func clockTicksToSeconds(ct uint64) time.Duration {
    s := float64(ct) / float64(c_SC_CLK_TCK)
    return time.Duration(float64(time.Second) * s)
}
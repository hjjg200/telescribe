package monitor

import (
    "fmt"
    "log"
    "strings"
)

// As /proc/meminfo uses kB, the functions here also return in kB

type procMeminfoStruct struct {
    memTotal int
    memFree int
    // memAvailable int // This is platform-specific
    buffers int
    cached int
    swapCached int
    swapTotal int
    swapFree int
}

var (
    procMeminfoMemTotal int
    procMeminfoSwapTotal int
)

func init() {

    // Get Total Memory
    pms, err := parseCurrentProcMeminfo()
    if err != nil {
        log.Fatalln(err)
    }
    procMeminfoMemTotal = pms.memTotal
    procMeminfoSwapTotal = pms.swapTotal

}

func parseCurrentProcMeminfo() (procMeminfoStruct, error) {
    cat, err := readFile("/proc/meminfo")
    if err != nil {
        return procMeminfoStruct{}, err
    }

    pms := procMeminfoStruct{}
    err = pms.Parse(cat)
    if err != nil {
        return procMeminfoStruct{}, err
    }

    return pms, nil
}

func(pms *procMeminfoStruct) Parse(meminfo string) error {

    var parseErr error
    n := 0 // Parsed item count
    lines := strings.Split(meminfo, "\n")

    for i := 0; i < len(lines) && parseErr == nil; i++ {

        cols := splitWhitespace(lines[i])
        getVal := func() int {
            var tmp int
            _, parseErr = fmt.Sscanf(cols[1], "%d", &tmp)
            return tmp
        }
        n += 1

        switch cols[0] {
        case "MemTotal:":
            pms.memTotal = getVal()
        case "MemFree:":
            pms.memFree = getVal()
        case "Buffers:":
            pms.buffers = getVal()
        case "Cached:":
            pms.cached = getVal()
        case "SwapCached:":
            pms.swapCached = getVal()
        case "SwapTotal:":
            pms.swapTotal = getVal()
        case "SwapFree:":
            pms.swapFree = getVal()
        default:
            n -= 1
        }

    }

    if parseErr != nil {
        return parseErr
    }
    if n != 7 {
        return fmt.Errorf("Bad /proc/meminfo")
    }

    return nil

}

func GetMemoryTotal() float64 {
    return float64(procMeminfoMemTotal)
}

func GetSwapTotal() float64 {
    return float64(procMeminfoSwapTotal)
}

// 

func GetMemoryFree() (float64, error) {
    pms, err := parseCurrentProcMeminfo()
    if err != nil {
        return 0, err
    }
    return float64(pms.memFree + pms.cached + pms.buffers), nil
}

func GetSwapFree() (float64, error) {
    pms, err := parseCurrentProcMeminfo()
    if err != nil {
        return 0, err
    }
    return float64(pms.swapFree + pms.swapCached), nil
}

func GetMemoryUsage() (float64, error) {
    t := GetMemoryTotal()
    f, err := GetMemoryFree()
    if err != nil {
        return 0.0, err
    }
    return (1.0 - (f / t)) * 100.0, nil
}

func GetSwapUsage() (float64, error) {
    t := GetSwapTotal()
    f, err := GetSwapFree()
    if err != nil {
        return 0.0, err
    }
    return (1.0 - (f / t)) * 100.0, nil
}
package monitor

import (
    "encoding/json"
    "fmt"
    "os"
    "syscall"
    "testing"
    "time"
)


func TestStatfs(t *testing.T) {

    statfs := func(path string) (syscall.Statfs_t, error) {
        fsbuf := &syscall.Statfs_t{}
        err := syscall.Statfs(path, fsbuf)
        return *fsbuf, err
    }

    fmt.Print("/"); fmt.Println(statfs("/"))
    for {

        for disk, parts := range devHierarchy {
            dev := "/dev/" + disk
            fmt.Print(dev + ": "); fmt.Println(statfs(dev))
            for _, part := range parts {
                dev := "/dev/" + part
                fmt.Print(dev + ": "); fmt.Println(statfs(dev))
            }
        }

        time.Sleep(3 * time.Second)

    }

}

func TestDiskIOTicks(t *testing.T) {

    // Test if io ticks = read ticks + write ticks
    parseDevStats()
    prev := parsedDevStats
    for {
        time.Sleep(1 * time.Second)

        parseDevStats()
        pds := parsedDevStats

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

func TestAllDevices(t *testing.T) {

    enc := json.NewEncoder(os.Stderr)
    enc.SetIndent("", "  ")

    for {

        out, _ := GetDevUsage("/code-server/shared")
        enc.Encode(out)
        enc.Encode(GetDevsSize())
        enc.Encode(GetDevsReads())
        fmt.Println("")

        time.Sleep(3 * time.Second)

    }

}
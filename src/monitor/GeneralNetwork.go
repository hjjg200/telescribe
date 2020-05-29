package monitor

import (
    "fmt"
    "strings"
    "sync/atomic"
    "time"

    "github.com/hjjg200/go-together"
)

// https://stackoverflow.com/questions/3521678/what-are-meanings-of-fields-in-proc-net-dev

type netStat struct {
    name       string
    in         int64 // Bytes
    out        int64
    inPackets  int64
    outPackets int64
}
var parsedNetStats map[string] netStat
var netRail = together.NewRailSwitch()
var netParsed int32
const netParseMinimumWait = time.Millisecond * 10

func parseNetworkStats() {

    netRail.Queue(railWrite, 1)
    defer netRail.Proceed(railWrite)

    if atomic.CompareAndSwapInt32(&netParsed, 0, 1) {

        netDev, err := readFile("/proc/net/dev")
        if err != nil {
            ErrorCallback(err)
        }

        pns := make(map[string] netStat)
        for _, line := range strings.Split(netDev, "\n") {
            var (
                name string
                in, inPackets, inErrs, inDrop, inFifo, inFrame, inCompressed, inMulticast int64
                out, outPackets, outErrs, outDrop, outFifo, outColls, outCarrier, outCompressed int64 
            )

            n, err := fmt.Sscanf(
                line, "%s %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
                &name, &in, &inPackets, &inErrs, &inDrop, &inFifo, &inFrame, &inCompressed, &inMulticast,
                &out, &outPackets, &outErrs, &outDrop, &outFifo, &outColls, &outCarrier, &outCompressed,
            )
            if n != 17 || err != nil {
                continue
            }

            // Remove colon from the name
            name = name[:len(name) - 1]

            pns[name] = netStat{
                name: name,
                in: in,
                inPackets: inPackets,
                out: out,
                outPackets: outPackets,
            }
        }

        parsedNetStats = pns

        go func() {
            time.Sleep(netParseMinimumWait)
            atomic.StoreInt32(&netParsed, 0)
        }()
    
    }

}

var prevNetworkIn = make(map[string] int64)
func GetNetworkIn() map[string] float64 {

    parseNetworkStats()
    netRail.Queue(railRead, 1)
    defer netRail.Proceed(railRead)

    pns := parsedNetStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, ok := prevNetworkIn[name]
        // Continue if prev is undefined
        if ok {ret[name] = float64(pns[name].in - prev)}
        prevNetworkIn[name] = pns[name].in
    }
    return ret

}

var prevNetworkInPackets = make(map[string] int64)
func GetNetworkInPackets() map[string] float64 {

    parseNetworkStats()
    netRail.Queue(railRead, 1)
    defer netRail.Proceed(railRead)

    pns := parsedNetStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, ok := prevNetworkInPackets[name]
        if ok {ret[name] = float64(pns[name].inPackets - prev)}
        prevNetworkInPackets[name] = pns[name].inPackets
    }
    return ret

}

var prevNetworkOut = make(map[string] int64)
func GetNetworkOut() map[string] float64 {

    parseNetworkStats()
    netRail.Queue(railRead, 1)
    defer netRail.Proceed(railRead)

    pns := parsedNetStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, ok := prevNetworkOut[name]
        if ok {ret[name] = float64(pns[name].out - prev)}
        prevNetworkOut[name] = pns[name].out
    }
    return ret

}

var prevNetworkOutPackets = make(map[string] int64)
func GetNetworkOutPackets() map[string] float64 {

    parseNetworkStats()
    netRail.Queue(railRead, 1)
    defer netRail.Proceed(railRead)

    pns := parsedNetStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, ok := prevNetworkOutPackets[name]
        if ok {ret[name] = float64(pns[name].outPackets - prev)}
        prevNetworkOutPackets[name] = pns[name].outPackets
    }
    return ret

}

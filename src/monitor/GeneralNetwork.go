package monitor

import (
    "fmt"
    "strings"
    "sync/atomic"
    "time"
)

// https://stackoverflow.com/questions/3521678/what-are-meanings-of-fields-in-proc-net-dev

type netStat struct {
    name string
    in int64 // Bytes
    inPackets int64
    out int64
    outPackets int64
}
var parsedNetworkStats map[string] netStat
const netStatParsingMinimumWait = time.Millisecond * 10
var netStatParsingWait int32

func init() {
    parseNetworkStats()

    // Initialize
    GetNetworkIn()
    GetNetworkInPackets()
    GetNetworkOut()
    GetNetworkOutPackets()
}

func parseNetworkStats() {

    if atomic.CompareAndSwapInt32(&netStatParsingWait, 0, 1) == false {
        // Already running
        return
    }

    netDev, err := readFile("/proc/net/dev")
    if err != nil {
        emitError(err)
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

    parsedNetworkStats = pns

    time.Sleep(netStatParsingMinimumWait)
    atomic.StoreInt32(&netStatParsingWait, 0)

}

var prevNetworkIn = make(map[string] int64)
func GetNetworkIn() map[string] float64 {
    parseNetworkStats()
    pns := parsedNetworkStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, _ := prevNetworkIn[name]
        ret[name] = float64(pns[name].in - prev)
        prevNetworkIn[name] = pns[name].in
    }
    return ret
}

var prevNetworkInPackets = make(map[string] int64)
func GetNetworkInPackets() map[string] float64 {
    parseNetworkStats()
    pns := parsedNetworkStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, _ := prevNetworkInPackets[name]
        ret[name] = float64(pns[name].inPackets - prev)
        prevNetworkInPackets[name] = pns[name].inPackets
    }
    return ret
}

var prevNetworkOut = make(map[string] int64)
func GetNetworkOut() map[string] float64 {
    parseNetworkStats()
    pns := parsedNetworkStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, _ := prevNetworkOut[name]
        ret[name] = float64(pns[name].out - prev)
        prevNetworkOut[name] = pns[name].out
    }
    return ret
}

var prevNetworkOutPackets = make(map[string] int64)
func GetNetworkOutPackets() map[string] float64 {
    parseNetworkStats()
    pns := parsedNetworkStats
    ret := make(map[string] float64)
    for name := range pns {
        prev, _ := prevNetworkOutPackets[name]
        ret[name] = float64(pns[name].outPackets - prev)
        prevNetworkOutPackets[name] = pns[name].outPackets
    }
    return ret
}

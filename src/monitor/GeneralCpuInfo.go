package monitor

import (
    "fmt"
    "log"
    "strings"
)

/*
And so, from fields listed in the first line of /proc/stat: (see section 1.8 at documentation)
     user    nice   system  idle      iowait irq   softirq  steal  guest  guest_nice
cpu  74608   2520   24433   1117073   6176   4054  0        0      0      0

Algorithmically, we can calculate the CPU usage percentage like:
PrevIdle = previdle + previowait
Idle = idle + iowait
PrevNonIdle = prevuser + prevnice + prevsystem + previrq + prevsoftirq + prevsteal
NonIdle = user + nice + system + irq + softirq + steal
PrevTotal = PrevIdle + PrevNonIdle
Total = Idle + NonIdle

# differentiate: actual value minus the previous one
totald = Total - PrevTotal
idled = Idle - PrevIdle
CPU_Percentage = (totald - idled)/totald
*/

type procStatStruct struct {
    name    string
    user    int
    nice    int
    system  int
    idle    int
    iowait  int
    irq     int
    softirq int
    steal   int
}

var (
    // 0 is the overall cpu usage
    // from 1 is per-core cpu usage
    procStatCpuCount int
    prevProcStat []procStatStruct
)

func init() {

    // Get CPU Count
    cat, err := readFile( "/proc/stat" )
    if err != nil {
        log.Fatalln( "Failed to initialize GeneralCpuUsage: failed to read /proc/stat" )
    }

    lines := strings.Split( cat, "\n" )
    if len( lines ) < 2 {
        log.Fatalln( "Failed to initialize GeneralCpuUsage: bad /proc/stat" )
    }

    procStatCpuCount = 0
    for i := 0; i < len( lines ); i++ {
        cols := splitWhitespace( lines[i] )
        if cols[0][:3] != "cpu" {
            break
        }
        if len( cols[0] ) > 3 {
            procStatCpuCount++
        }
    }

    // Init Vars
    prevProcStat = make( []procStatStruct, int( GetCpuCount() ) + 1 )

}

func( pss procStatStruct ) GetTotal() int {
    return pss.user + pss.nice + pss.system + pss.idle + pss.iowait + pss.irq + pss.softirq + pss.steal
}

func( pss procStatStruct ) GetIdle() int {
    return pss.idle + pss.iowait
}

func( pss *procStatStruct ) Parse( line string ) error {

    n, err := fmt.Sscanf(
        line,
        "%s %d %d %d %d %d %d %d %d",
        &pss.name, &pss.user, &pss.nice, &pss.system, &pss.idle, &pss.iowait, &pss.irq, &pss.softirq, &pss.steal,
    )
    if n != 9 || err != nil {
        return fmt.Errorf( "Failed to read cpu stat" )
    }
    return nil

}

// GetCpuUsage returns the cpu usage during the duration from the last call of this function of the start of the current session
// and to the current call of this function.

func GetCpuUsage() ( []float64, error ) {

    cat, err := readFile( "/proc/stat" )
    if err != nil {
        return nil, err
    }

    cpup1 := int( GetCpuCount() ) + 1
    lines := strings.Split( cat, "\n" )
    lines = lines[:cpup1]

    usage := make( []float64, cpup1 )
    currProcStat := make( []procStatStruct, cpup1 )
    for i := 0; i < len( lines ); i++ {

        // Get Current Values
        currProcStat[i] = procStatStruct{}
        err = currProcStat[i].Parse( lines[i] )
        if err != nil {
            return nil, err
        }

        // Get Difference
        dIdle := float64( currProcStat[i].GetIdle() - prevProcStat[i].GetIdle() )
        dTotal := float64( currProcStat[i].GetTotal() - prevProcStat[i].GetTotal() )

        // Get Usage
        usage[i] = ( dTotal - dIdle ) / dTotal * 100

    }

    // Assign Previous Value
    prevProcStat = currProcStat
    return usage, nil
}

// GetCpuCount returns the count of the cpu from the /proc/stat

func GetCpuCount() float64 {
    return float64( procStatCpuCount )
}
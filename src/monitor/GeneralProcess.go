package monitor

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync/atomic"
    "time"

    "github.com/hjjg200/go-together"
)

type pidStatStruct struct {
    pid       int    // 1
    comm      string // 2
    state     byte   // 3  %c 
    ppid      int    // 4  %d
    utime     uint64 // 14 %lu
    stime     uint64 // 15 %lu
    cutime    int64  // 16 %ld
    cstime    int64  // 17 %ld
    starttime uint64 // 22 %llu
}

const (
    processRunning  byte = 'R'
    processSleeping byte = 'S'
    processWaiting  byte = 'D'
    processStopped  byte = 'T'
    processZombie   byte = 'Z'
)

type pidIoStruct struct {
    readBytes  uint64
    writeBytes uint64
}

type pidSmapsStruct struct {
    uss  uint64 // Private_Clean + Private_Dirty
    swap uint64 // Swap
}

var pidArg0s       = make(map[int] string)
var parsedPidStats = make(map[int] pidStatStruct)
var parsedPidIos   = make(map[int] pidIoStruct)
var parsedPidSmaps = make(map[int] pidSmapsStruct)

var processRail = together.NewRailSwitch()
var processParsed int32
const processParseMinimumWait = time.Second * 1

func init() {
    parseProcesses()
}

func parseProcesses() {

    processRail.Queue(railWrite, 1)
    defer processRail.Proceed(railWrite)
    if atomic.CompareAndSwapInt32(&processParsed, 0, 1) {

        matches, err := filepath.Glob("/proc/*")
        if err != nil {
            panic(err)
        }
        
        // Cleanup
        pidArg0s = make(map[int] string)
        parsedPidStats = make(map[int] pidStatStruct)
        parsedPidIos   = make(map[int] pidIoStruct)
        parsedPidSmaps = make(map[int] pidSmapsStruct)

        EachProcess:
        for _, match := range matches {

            var pid int
            n, err := fmt.Sscanf(match, "/proc/%d", &pid)
            if n != 1 || err != nil {
                continue
            }

            // string
            spid := fmt.Sprint(pid)

            // /proc/[pid]/stat
            stat, err := readFile("/proc/" + spid + "/stat")
            switch {
            case os.IsPermission(err): // Ignore unaccessible processes
                continue EachProcess
            case err != nil:
                ErrorCallback(err)
                continue EachProcess
            }
            ppids := pidStatStruct{}
            err = ppids.Parse(stat)
            if err != nil {
                ErrorCallback(err)
                continue
            }

            // arg0 preference:
            // 1. resolved link of /proc/[pid]/exe 
            // 2. arg 0 of /proc/[pid]/cmdline
            // 3. comm of /proc/[pid]/stat

            // /proc/[pid]/exe
            // Under Linux 2.2 and later, this file is a symbolic link con‐
            // taining the actual pathname of the executed command.
            arg0, err := filepath.EvalSymlinks("/proc/" + spid + "/exe")
            if err != nil {

                // /proc/[pid]/cmdline
                // On fail use arg0 of cmdline instead

                cmdline, err := readFile("/proc/" + spid + "/cmdline")
                switch {
                case os.IsPermission(err): // Ignore unaccessible processes
                    continue EachProcess
                case err != nil:
                    ErrorCallback(err)
                    continue EachProcess
                }

                // cmdline is separated by null chars
                arg0 = strings.SplitN(cmdline, "\x00", 2)[0]
                if len(arg0) == 0 {
                    // In case of empty cmdline, use comm
                    arg0 = ppids.GetComm()
                }

            }

            // /proc/[pid]/io
            io, err := readFile("/proc/" + spid + "/io")
            switch {
            case os.IsPermission(err): // Ignore unaccessible processes
                continue EachProcess
            case err != nil:
                ErrorCallback(err)
                continue EachProcess
            }

            ppidio := pidIoStruct{}
            err = ppidio.Parse(io)
            if err != nil {
                ErrorCallback(err)
                continue
            }

            // /proc/[pid]/smaps
            smaps, err := readFile("/proc/" + spid + "/smaps")
            switch {
            case os.IsPermission(err): // Ignore unaccessible processes
                continue EachProcess
            case err != nil:
                ErrorCallback(err)
                continue EachProcess
            }

            ppidmp := pidSmapsStruct{}
            err = ppidmp.Parse(smaps)
            if err != nil {
                ErrorCallback(err)
                continue
            }

            // Assign
            pidArg0s[pid] = arg0
            parsedPidStats[pid] = ppids
            parsedPidIos[pid]   = ppidio
            parsedPidSmaps[pid] = ppidmp

        }

        go func() {
            time.Sleep(processParseMinimumWait)
            atomic.StoreInt32(&processParsed, 0)
        }()
    
    }

}

// Must be wrapped inside RLock when called
func getProcessIds(key string) []int {

    // Check for pid
    var pid int
    n, _ := fmt.Sscanf(key, "%d", &pid)
    if _, ok := parsedPidStats[pid]; n == 1 && ok {
        return []int{pid}
    }

    // Check for comm and arg0
    pids := []int{}
    for pid, ppids := range parsedPidStats {
        if ppids.GetComm() == key {
            pids = append(pids, pid)
        }

        fp0      := pidArg0s[pid]
        fp1, err := filepath.EvalSymlinks(key)
        if fp0 == fp1 && err == nil {
            pids = append(pids, pid)
        }
    }

    return pids

}

// CPU and parent process
// proc/[pid]/cmdline
//          This read-only file holds the complete command line for the
//          process, unless the process is a zombie.  In the latter case,
//          there is nothing in this file: that is, a read on this file
//          will return 0 characters.  The command-line arguments appear
//          in this file as a set of strings separated by null bytes
//          ('\0'), with a further null byte after the last string.
//
// /proc/[pid]/statm
//          Provides information about memory usage, measured in pages.
//          The columns are:
//
//              size       (1) total program size
//                           (same as VmSize in /proc/[pid]/status)
//              resident   (2) resident set size
//                           (same as VmRSS in /proc/[pid]/status)
//              shared     (3) number of resident shared pages (i.e., backed by a file)
//                           (same as RssFile+RssShmem in /proc/[pid]/status)
//              text       (4) text (code)
//              lib        (5) library (unused since Linux 2.6; always 0)
//              data       (6) data + stack
//              dt         (7) dirty pages (unused since Linux 2.6; always 0)
//
// /proc/[pid]/stat
// (3) state  %c
//          One of the following characters, indicating process
//            state:
//
//          R  Running
//
//          S  Sleeping in an interruptible wait
//
//          D  Waiting in uninterruptible disk sleep
//
//          Z  Zombie = exited
//
//          T  Stopped (on a signal) or (before Linux 2.6.33)
//                trace stopped; SIGCONT wakes it and SIGKILL kills it
//
//          R, S, D, T = still running
//          Z = terminated
//
// (14) utime  %lu
//         Amount of time that this process has been scheduled
//         in user mode, measured in clock ticks (divide by
//         sysconf(_SC_CLK_TCK)).  This includes guest time,
//         guest_time (time spent running a virtual CPU, see
//         below), so that applications that are not aware of
//         the guest time field do not lose that time from
//         their calculations.
//
// (15) stime  %lu
//         Amount of time that this process has been scheduled
//         in kernel mode, measured in clock ticks (divide by
//         sysconf(_SC_CLK_TCK)).
//
// (16) cutime  %ld
//          Amount of time that this process's waited-for chil‐
//          dren have been scheduled in user mode, measured in
//          clock ticks (divide by sysconf(_SC_CLK_TCK)).  (See
//          also times(2).)  This includes guest time,
//          cguest_time (time spent running a virtual CPU, see
//          below).
//
// (17) cstime  %ld
//          Amount of time that this process's waited-for chil‐
//          dren have been scheduled in kernel mode, measured in
//          clock ticks (divide by sysconf(_SC_CLK_TCK)).
//
// total_time = utime + stime
// children_total_time = cutime + cstime
// grand_total_time = total_time + children_total_time
//
// seconds = now - last eval time
// usage = 100.0 * (total_time / _SC_CLK_TCK) / seconds

var prevProcessCpuGroupCpuTime = make(map[string] int) // Entire cpu time
var prevProcessCpuPidStats     = make(map[int] pidStatStruct)
var prevProcessCpuArg0s        = make(map[int] string)
func GetProcessCpuUsage(key string) (float64, error) {

    parseProcesses()
    processRail.Queue(railRead, 1)
    defer processRail.Proceed(railRead)

    pids := getProcessIds(key)
    if len(pids) == 0 {
        return 0.0, fmt.Errorf("Process not found")
    }

    // CPU time is per group in order to prevent percentage reflecting wrong value
    prevCpuTime, groupOk := prevProcessCpuGroupCpuTime[key] // defaults to 0 when key doesn't exist
    cpuTime              := getProcStats()[0].GetTotal()
    pastCpuTime          := cpuTime - prevCpuTime
    
    // Set previous variables
    prevProcessCpuGroupCpuTime[key] = cpuTime

    // Variables
    initialized := false && groupOk // Check group is parsed before
    ret         := 0.0

    for _, pid := range pids {

        ppids             := parsedPidStats[pid]
        prevPpids, prevOk := prevProcessCpuPidStats[pid]
        arg0              := pidArg0s[pid]
        prevArg0, _       := prevProcessCpuArg0s[pid]

        // Set previous variables
        prevProcessCpuPidStats[pid] = ppids
        prevProcessCpuArg0s[pid]    = arg0

        switch {
        case !prevOk,          // first parse
            prevArg0 != arg0: // pid owner changed
            
            continue // Uninitialized
        }

        initialized = true

        // Metrics
        ownTotal     := ppids.GetOwnTotal()
        prevOwnTotal := prevPpids.GetOwnTotal()
        totalDiff    := ownTotal - prevOwnTotal

        ret += float64(totalDiff) / float64(pastCpuTime) * 100.0

    }

    if !initialized {return 0.0, fmt.Errorf("Not initialized")}
    return ret, nil

}

func(ppids pidStatStruct) GetComm() string { // unwrap parenthesis
    if len(ppids.comm) < 2 {
        return ""
    }
    return ppids.comm[1:len(ppids.comm) - 1]
}

func(ppids pidStatStruct) GetOwnTotal() uint64 {
    return ppids.utime + ppids.stime
}

func(ppids pidStatStruct) GetChildrenTotal() uint64 {
    return uint64(ppids.cutime + ppids.cstime)
}

func(ppids pidStatStruct) GetTotal() uint64 {
    return ppids.GetOwnTotal() + ppids.GetChildrenTotal()
}

func(ppids *pidStatStruct) Parse(line string) error {

    var (
        pgrp, session, tty_nr, tpgid int // %d - 5, 6, 7, 8
        flags uint // %u - 9
        minflt, cminflt, majflt, cmajflt uint64 // %lu - 10, 11, 12, 13
        priority, nice, num_threads, iteralvalue uint // %ld - 18, 19, 20, 21
    )
    n, err := fmt.Sscanf(
        line,
        //        04 05       08 09 10       13 14    16 17 18       21 22
        "%d %s %c %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
        &ppids.pid, &ppids.comm, &ppids.state, &ppids.ppid, // 1 - 4
        &pgrp, &session, &tty_nr, &tpgid, // 5 - 8
        &flags, // 9
        &minflt, &cminflt, &majflt, &cmajflt, // 10 - 13
        &ppids.utime, &ppids.stime, &ppids.cutime, &ppids.cstime, // 14 - 17
        &priority, &nice, &num_threads, &iteralvalue, // 18 - 21
        &ppids.starttime, // 22
   )
    if n != 22 || err != nil {
        return fmt.Errorf("Failed to read proc pid stat")
    }
    return nil

}

// IO usage
// /proc/[pid]/io

// https://unix.stackexchange.com/questions/335809/do-the-figures-on-proc-pid-io-include-paging-and-swapping-i-o

// rchar and wchar may be not accurate as, for rchar as example,
// "It includes things such as terminal I/O and is
// unaffected by whether or not actual physical disk I/O
// was required (the read might have been satisfied from
// pagecache)."

const (
    typeProcessIoReadBytes = iota
    typeProcessIoWriteBytes
)

var prevProcessIoValues = make(map[int/* type */] map[int/* pid */] uint64)
var prevProcessIoArg0s  = make(map[int/* type */] map[int/* pid */] string)
func getProcessIo(key string, typ int) (float64, error) {

    processRail.Queue(railRead, 1)
    defer processRail.Proceed(railRead)

    if _, ok := prevProcessIoValues[typ]; !ok {
        prevProcessIoValues[typ] = make(map[int] uint64)
        prevProcessIoArg0s[typ]  = make(map[int] string)
    }

    pids := getProcessIds(key)
    if len(pids) == 0 {
        return 0.0, fmt.Errorf("Process not found")
    }

    initialized := false
    ret         := 0.0
    for _, pid := range pids {
        
        ppidio := parsedPidIos[pid]
        var value uint64
        switch typ {
        case typeProcessIoReadBytes:  value = ppidio.readBytes
        case typeProcessIoWriteBytes: value = ppidio.writeBytes
        }

        prevValue, prevOk := prevProcessIoValues[typ][pid]
        arg0              := pidArg0s[pid]
        prevArg0, _       := prevProcessIoArg0s[typ][pid]

        // Set previous vars
        prevProcessIoValues[typ][pid] = value
        prevProcessIoArg0s[typ][pid]  = arg0

        switch {
        case !prevOk,         // first parse
            prevArg0 != arg0: // pid owner changed

            continue // Uninitialized
        }

        initialized = true
        ret += float64(value - prevValue)

    }

    if !initialized {return 0.0, fmt.Errorf("Not initialized")}
    return ret, nil

}

func GetProcessReadBytes(key string) (float64, error) {
    parseProcesses()
    return getProcessIo(key, typeProcessIoReadBytes)
}

func GetProcessWriteBytes(key string) (float64, error) {
    parseProcesses()
    return getProcessIo(key, typeProcessIoWriteBytes)
}

func(ppidio *pidIoStruct) Parse(io string) error {

    For:
    for _, line := range strings.Split(io, "\n") {

        var ptr interface{}
        splits := splitWhitespace(line)

        switch splits[0] {
        case "read_bytes:":  ptr = &ppidio.readBytes
        case "write_bytes:": ptr = &ppidio.writeBytes
        default:
            continue For
        }

        n, err := fmt.Sscanf(splits[1], "%d", ptr)
        if n != 1 || err != nil {
            return fmt.Errorf("Bad proc/[pid]/io: %s", err.Error())
        }

    }

    return nil

}

// Memory usage
// /proc/[pid]/smaps
//
// https://unix.stackexchange.com/questions/33381/getting-information-about-a-process-memory-usage-from-proc-pid-smaps

// https://selenic.com/repo/smem/file/tip/smem
// From smem's source code:
//
// uss=('USS', lambda n: pt[n]['private_clean']
//      + pt[n]['private_dirty'], '% 8a', sum,
//      'unique set size'),
// rss=('RSS', lambda n: pt[n]['rss'], '% 8a', sum,
//      'resident set size (ignoring sharing)'),
// pss=('PSS', lambda n: pt[n]['pss'], '% 8a', sum,
//      'proportional set size (including sharing)'),
// vss=('VSS', lambda n: pt[n]['size'], '% 8a', sum,
//      'virtual set size (total virtual address space mapped)'),
//
// USS = Private_Clean + Private_Dirty

const (
    typeProcessSmapMemory = iota
    typeProcessSmapSwap
)

func getProcessSmap(key string, typ int) (float64, error) {

    processRail.Queue(railRead, 1)
    defer processRail.Proceed(railRead)

    pids := getProcessIds(key)
    if len(pids) == 0 {
        return 0.0, fmt.Errorf("Process not found")
    }

    total := uint64(0)
    for _, pid := range pids {
        ppidmp := parsedPidSmaps[pid]
        switch typ {
        case typeProcessSmapMemory: total += ppidmp.uss
        case typeProcessSmapSwap:   total += ppidmp.swap
        }
    }

    // Divide
    ret := float64(total)
    switch typ {
    case typeProcessSmapMemory: ret /= GetMemoryTotal()
    case typeProcessSmapSwap:   ret /= GetSwapTotal()
    }
    
    return ret * 100.0, nil

}

func GetProcessMemoryUsage(key string) (float64, error) {
    parseProcesses()
    return getProcessSmap(key, typeProcessSmapMemory)
}

func GetProcessSwapUsage(key string) (float64, error) {
    parseProcesses()
    return getProcessSmap(key, typeProcessSmapSwap)
}

func(ppidmp *pidSmapsStruct) Parse(smaps string) error {

    if len(smaps) == 0 {
        return fmt.Errorf("Bad smaps")
    }

    *ppidmp = pidSmapsStruct{} // Reset

    i := 0
    ParseLoop:
    for {

        // Vars
        name  := ""
        vstr  := ""
        value := uint64(0)
        
        // For parsing
        kvrow        := false
        reachedColon := false
        inWhitespace := false
        col := 1
        
        RowParse:
        for {

            // Check position
            if i == len(smaps) {
                break ParseLoop
            }

            // Token
            token := smaps[i]
            i += 1
            switch token {
            case ' ', '\t': // whitespaces
                if !reachedColon { // not a key value row
                    break RowParse
                }
                if !inWhitespace {
                    inWhitespace = true
                    if col == 2 { // value is read
                        kvrow = true
                        break RowParse
                    }
                }
            case ':': // colon
                if !reachedColon {
                    reachedColon = true
                }
            case '\n': // line break
                break RowParse
            default:
                if inWhitespace {
                    col += 1
                    inWhitespace = false
                }

                if col == 1 {
                    name += string(token)
                } else if col == 2 {
                    vstr += string(token)
                }
            }

        }

        // Key-value row
        if !kvrow {
            continue
        }

        // Value
        var err error
        value, err = strconv.ParseUint(vstr, 10, 64)
        if err != nil {
            // string values
            continue
        }

        // Name
        switch name {
        case "Private_Clean", "Private_Dirty":
            ppidmp.uss += value
        case "Swap":
            ppidmp.swap += value
        }

    }

    return nil

}
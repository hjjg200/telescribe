package monitor

import (
    "fmt"
    "path/filepath"
    "sync"
    "sync/atomic"
    "time"
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

var pidArg0s = make(map[int] string)
var parsedPidStats = make(map[int] pidStatStruct)
var processMutex sync.RWMutex
var processParsed int32
const processParseMinimumWait = time.Second * 1

func init() {
    parseProcesses()
}

func parseProcesses() {

    processMutex.Lock()
    defer processMutex.Unlock()
    if atomic.CompareAndSwapInt32(&processParsed, 0, 1) {

        matches, err := filepath.Glob("/proc/*")
        if err != nil {
            panic(err)
        }

        for _, match := range matches {

            var pid int
            n, err := fmt.Sscanf(match, "/proc/%d", &pid)
            if n != 1 || err != nil {
                continue
            }

            // string
            spid := fmt.Sprint(pid)

            // /proc/[pid]/exe
            // Under Linux 2.2 and later, this file is a symbolic link con‐
            // taining the actual pathname of the executed command.
            arg0, err := filepath.EvalSymlinks("/proc/" + spid + "/exe")
            if err != nil {
                ErrorCallback(err)
                continue
            }

            // /proc/[pid]/stat
            stat, err := readFile("/proc/" + spid + "/stat")
            if err != nil {
                ErrorCallback(err)
                continue
            }
            ppids := pidStatStruct{}
            err = ppids.Parse(stat)
            if err != nil {
                ErrorCallback(err)
                continue
            }

            // Assign
            pidArg0s[pid] = arg0
            parsedPidStats[pid] = ppids

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


var prevCpuUsageArg0s = make(map[int] string)
var prevCpuUsagePidStats = make(map[int] pidStatStruct)
var prevCpuUsageCpuTime = make(map[int] int) // Entire cpu
var prevCpuUsageGroupCpuTime = make(map[string] int) // Entire cpu time
func GetProcessCpuUsage(key string) (float64, error) {

    parseProcesses()
    processMutex.RLock()
    defer processMutex.RUnlock()

    pids := getProcessIds(key)
    if len(pids) == 0 {
        return 0.0, fmt.Errorf("Process not found")
    }

    ret := 0.0

    // CPU time is per group in order to prevent percentage reflecting wrong value
    prevCpuTime, _ := prevCpuUsageGroupCpuTime[key] // defaults to 0 when key doesn't exist
    cpuTime        := getProcStats()[0].GetTotal()
    pastCpuTime    := cpuTime - prevCpuTime
    
    prevCpuUsageGroupCpuTime[key] = cpuTime

    for _, pid := range pids {

        ppids := parsedPidStats[pid]

        // Parse
        var prevPpids   pidStatStruct
        arg0         := pidArg0s[pid]
        prevArg0, ok := prevCpuUsageArg0s[pid]
        switch {
        case !ok,             // first parse
            prevArg0 != arg0: // pid owner changed
            // default to zero
        default: // was parsed before
            prevPpids   = prevCpuUsagePidStats[pid]
        }

        // Prev
        prevCpuUsageArg0s[pid] = arg0

        // Metrics
        ownTotal     := ppids.GetOwnTotal()
        prevOwnTotal := prevPpids.GetOwnTotal()
        totalDiff    := ownTotal - prevOwnTotal

        ret += float64(totalDiff) / float64(pastCpuTime) * 100.0

    }

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

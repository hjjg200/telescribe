package monitor

type pidStatStruct struct {
    pid    int     // 1
    comm   string  // 2
    state  byte    // 3  %c 
    ppid   int     // 4  %d
    utime  uint64  // 14 %lu
    stime  uint64  // 15 %lu
    cutime int64   // 16 %ld
    cstime int64   // 17 %ld
}

const (
    processRunning  byte = 'R'
    processSleeping byte = 'S'
    processWaiting  byte = 'D'
    processStopped  byte = 'T'
    processZombie   byte = 'Z'
)


// CPU and parent process
// /proc/[pid]/stat

//(3) state  %c
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

func(ppids pidStatStruct) GetComm() string { // unwrap parenthesis
    if len(ppids) < 2 {
        return ""
    }
    return ppids.comm[1:len(ppids.comm) - 1]
}

func(ppids pidStatStruct) GetOwnTotal() int {
    return ppids.utime + ppids.stime
}

func(ppids pidStatStruct) GetChildrenTotal() int {
    return ppids.cutime + ppids.cstime
}

func(ppids pidStatStruct) GetTotal() int {
    return ppids.GetOwnTotal() + ppids.GetChildrenTotal()
}

func(ppids pidStatStruct) Parse(line string) error {

    var (
        pgrp, session, tty_nr, tpgid int // %d - 5, 6, 7, 8
        flags uint // %u - 9
        minflt, cminflt, majflt, cmajflt uint64 // %lu - 10, 11, 12, 13
    )
    n, err := fmt.Sscanf(
        line,
        //        04 05       08 09 10       13 14    16 17
        "%d %s %c %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
        &ppids.pid, &ppids.comm, &ppids.state, &ppids.ppid, // 1 - 4
        pgrp, session, tty_nr, tpgid, // 5 - 8
        flags, // 9
        minflt, cminflt, majflt, cmajflt, // 10 - 13
        &ppids.utime, &ppids.stime, &ppids.cutime, &ppids.cstime, // 14 - 17
   )
    if n != 17 || err != nil {
        return fmt.Errorf("Failed to read proc pid stat")
    }
    return nil

}

// IO usage
// /proc/[pid]/io

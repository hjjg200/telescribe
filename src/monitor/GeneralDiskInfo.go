package monitor

// /etc/mtab
// df -kP
// When both the -k and -P options are specified, the following header line shall be written (in the POSIX locale):
// "Filesystem 1024-blocks Used Available Capacity Mounted on\n"
// When the -P option is specified without the -k option, the following header line shall be written (in the POSIX locale):
// "Filesystem 512-blocks Used Available Capacity Mounted on\n"

// https://unix.stackexchange.com/questions/60091/where-is-the-documentation-for-what-sda-sdb-dm-0-dm-1-mean
// https://unix.stackexchange.com/questions/4561/how-do-i-find-out-what-hard-disks-are-in-the-system
// https://stackoverflow.com/questions/37248948/how-to-get-disk-read-write-bytes-per-second-from-proc-in-programming-on-linux

// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-block


// Reference source codes:
// https://github.com/coreutils/coreutils/blob/master/src/df.c
// https://github.com/coreutils/gnulib/blob/master/lib/mountlist.c
// https://code.woboq.org/userspace/glibc/sysdeps/unix/sysv/linux/statvfs.c.html#39
// https://github.com/coreutils/gnulib/blob/e93bdf684a7ed2c9229c0f113e33dc639f1050d7/lib/mountlist.c#L444

// statfs
// https://www.man7.org/linux/man-pages/man2/statfs.2.html

// Use /proc/self/mountinfo to get mounted disks

// df uses statfs syscall to get disk informations and
// for path parameter, it uses target and disk in order of preference to get info
// and the target and disk info is acquired from /proc/self/mountinfo

// lsblk
// https://github.com/karelzak/util-linux/blob/master/misc-utils/lsblk.c
// lsblk checks device types by sysfs device/type
// https://github.com/karelzak/util-linux/blob/498f910eeb0167c75b0421a9a12974e6bb0afb04/lib/blkdev.c#L325
// and it states a device is partition if its sysfs has no children


// OVERVIEW ---
// Get all entries from /proc/partitions
// those entries will be in /sys/block
// check if they are mounted by /proc/self/mountinfo
// statfs with the mount point
// e.g., disk-usage(<devname/uuid/label/id/mount point>) => statfs
// e.g., disk-io-usage... => /sys/block

import (
    "fmt"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
    "time"

    . "github.com/hjjg200/go-act"
)

type devStatStruct struct {
    reads        int64
    readSectors  int64 // 1 sector is 512 bytes
    readTicks    int64 // millisecond
    writes       int64
    writeSectors int64
    writeTicks   int64
    ioTicks      int64 // millisecond
}

type statfsStruct struct {
    blocksize uint64
    blocks    uint64
    free      uint64
}

type mountInfoStruct struct {
    mountId int // 1
    parentId int // 2
    majorMinor string // 3
    root string // 4
    target string // 5
    fsType string // 9
    source string // 10
}

const devParseMinimumWait = time.Millisecond * 10
var devMutex sync.RWMutex
var devParsed int32

var devHierarchy map[string] []string
var devAliases map[string] []string
var devTargets map[string] []string
var devMajorMinors map[string] string

var parsedDevStats   map[string] devStatStruct
var parsedStatfs     map[string] statfsStruct
var parsedMountInfos []mountInfoStruct

func init() {
    parseDevStats()
}

func parseDevHierarchy() {

    // Partitions
    pt, err := readFile("/proc/partitions")
    if err != nil {
        ErrorCallback(err)
        return
    }

    // Cleanup
    devAliases = make(map[string] []string)

    // Aliases
    byDirs, err := filepath.Glob("/dev/by-*")
    if err == nil {
        for _, byDir := range byDirs {
            
            entries, err := filepath.Glob(byDir + "/*")
            if err != nil {
                continue
            }

            for _, entry := range entries {

                eval, err := filepath.EvalSymlinks(entry)
                if err != nil {
                    continue
                }

                dev := filepath.Base(eval)
                if _, ok := devAliases[dev]; !ok {
                    devAliases[dev] = []string{}
                }
                devAliases[dev] = append(
                    devAliases[dev], filepath.Base(entry),
                )

            }

        }

    }

    // Mountinfo
    // https://github.com/coreutils/gnulib/blob/a2080f6506701d8d9ca5111d628607a6a8013f61/lib/mountlist.c#L469
    parsedMountInfos  = make([]mountInfoStruct, 0)
    procmi, err      := readFile("/proc/self/mountinfo")
    if err != nil {
        ErrorCallback(err)
        return
    }
    for _, line := range strings.Split(procmi, "\n") {func() {
        
        defer func() {
            if r := recover(); r != nil {
                ErrorCallback(fmt.Errorf("%v", r))
            }
        }()

        next := func(f string, term string, ptr interface{}) error {

            i := strings.Index(line, term)
            if i == -1 {return nil}
            sub  := line
            line  = line[i + len(term):]

            if ptr == nil {return nil}
            n, err := fmt.Sscanf(sub, f, ptr)
            if n != 1     {return fmt.Errorf("Failed sscanf")}
            if err != nil {return err}

            return nil

        }

        mi := mountInfoStruct{}
        Try(next("%d", " ", &mi.mountId))
        Try(next("%d", " ", &mi.parentId))
        Try(next("%s", " ", &mi.majorMinor))
        Try(next("%s", " ", &mi.root))
        Try(next("%s", " ", &mi.target))
        Try(next("", " ",   nil)) // ignore options
        Try(next("", "- ", nil)) // ignore optional fields
        Try(next("%s", " ", &mi.fsType))
        Try(next("%s", " ", &mi.source))

        parsedMountInfos = append(parsedMountInfos, mi)

    }()}

    // Device hierarchy
    devHierarchy   = make(map[string] []string)
    devTargets     = make(map[string] []string)
    devMajorMinors = make(map[string] string)
    for _, line := range strings.Split(pt, "\n") {

        var (
            major, minor, blocks int64
            name string
        )
        n, err := fmt.Sscanf(line, "%d %d %d %s", &major, &minor, &blocks, &name)
        if err != nil || n != 4 {
            continue
        }

        if name[:2] == "md"   {continue}
        if name[:4] == "loop" {continue}

        devHierarchy[name]   = make([]string, 0)
        devMajorMinors[name] = fmt.Sprintf("%d:%d", major, minor)

        // Check mounted
        for _, mi := range parsedMountInfos {
            if mi.source == "/dev/" + name {
                if _, ok := devTargets[name]; !ok {devTargets[name] = []string{}}
                devTargets[name] = append(devTargets[name], mi.target)
            }
        }

    }

    // Check partitions
    PartitionParseLoop:
    for dev := range devHierarchy {
        for dev2 := range devHierarchy {
            if dev == dev2 { continue }

            if strings.HasPrefix(dev, dev2) {
                // dev is partition of dev2
                // remove partition from disk lists
                delete(devHierarchy, dev)
                // Add partition
                devHierarchy[dev2] = append(devHierarchy[dev2], dev)
                continue PartitionParseLoop
            }
        }
    }

}

func getDev(key string) string {

    allDevs := []string{}
    for disk, parts := range devHierarchy {
        allDevs = append(allDevs, disk)
        allDevs = append(allDevs, parts...)
    }

    for _, dev := range allDevs {
        if dev == key {return dev}
    }

    for dev, aliases := range devAliases {
        if dev == key {
            return dev
        }
        for _, alias := range aliases {
            if alias == key {
                return dev
            }
        }
    }

    for dev, targets := range devTargets {
        for _, target := range targets {
            if target == key {
                return dev
            }
        }
    }

    return ""

}

func getParentDisk(part string) string {

    for disk, parts := range devHierarchy {
        for _, part2 := range parts {
            if part == part2 {
                return disk
            }
        }
    }
    return ""

}

func parseDevStats() {

    devMutex.Lock()
    defer devMutex.Unlock()

    if atomic.CompareAndSwapInt32(&devParsed, 0, 1) {

        // Prepare
        parseDevHierarchy()
        parsedStatfs = make(map[string] statfsStruct)

        // Statfs
        for dev, targets := range devTargets {

            var ok bool
            var statfs syscall.Statfs_t
            for _, target := range targets {
                var buf syscall.Statfs_t
                err := syscall.Statfs(target, &buf)
                if err == nil {
                    statfs = buf
                    ok = true
                    break
                }
            }
            if !ok {
                continue
            }

            // Many space usage primitives use all 1 bits to denote a value that is
            // not applicable or unknown.
            var blocksize uint64
            if ^statfs.Frsize == 0 || statfs.Frsize == 0 {
                if ^statfs.Bsize == 0 {continue}
                blocksize = uint64(statfs.Bsize)
            } else {
                blocksize = uint64(statfs.Frsize)
            }

            // Assign
            parsedStatfs[dev] = statfsStruct{
                blocksize: blocksize,
                blocks: statfs.Blocks,
                free: statfs.Bfree,
            }

        }

        // Per device
        allDevs := make([]string, 0)
        for parent, children := range devHierarchy {
            allDevs = append(allDevs, parent)
            allDevs = append(allDevs, children...)
        }
        parsedDevStats = make(map[string] devStatStruct)
        for _, dev := range allDevs {

            var (
                reads, readMerges, readSectors, readTicks int64
                writes, writeMerges, writeSectors, writeTicks int64
                inFlight, ioTicks, timeInQueue int64
            )
            
            majorMinor := devMajorMinors[dev]
            st, err    := readFile(fmt.Sprintf("/sys/dev/block/%s/stat", majorMinor))
            if err != nil {
                continue
            }

            n, err := fmt.Sscanf(
                st, "%d %d %d %d %d %d %d %d %d %d %d",
                &reads, &readMerges, &readSectors, &readTicks,
                &writes, &writeMerges, &writeSectors, &writeTicks,
                &inFlight, &ioTicks, &timeInQueue,
            )
            if n != 11 || err != nil {
                continue
            }

            parsedDevStats[dev] = devStatStruct{
                reads: reads,
                readSectors: readSectors,
                readTicks: readTicks,
                writes: writes,
                writeSectors: writeSectors,
                writeTicks: writeTicks,
                ioTicks: ioTicks,
            }

        }

        go func() {
            time.Sleep(devParseMinimumWait)
            atomic.StoreInt32(&devParsed, 0)
        }()

    }

}

// STAT ---

const (
    typeDevReads = iota
    typeDevWrites
    typeDevReadBytes
    typeDevWriteBytes
)

var prevDevStats = make(map[int] map[string] int64)
func getDevStat(key string, typ int, nested bool) interface{} {

    if !nested {
        parseDevStats()
        devMutex.RLock()
        defer devMutex.RUnlock()
    }

    // Multiple
    if key == "*" {
        ret := make(map[string] float64)
        for dev := range parsedDevStats {
            out := getDevStat(dev, typ, true)
            if out == nil {continue}
            ret[dev] = out.(float64)
        }
        if len(ret) == 0 {return nil}
        return ret
    }
    
    // Single
    dev := getDev(key)
    if dev == "" {return nil}

    if _, ok := prevDevStats[typ]; !ok {
        prevDevStats[typ] = make(map[string] int64)
    }

    var curr int64
    prev, prevOk    := prevDevStats[typ][dev]
    devStat, statOk := parsedDevStats[dev]
    if !statOk {return nil}

    switch typ {
    case typeDevReads: curr = devStat.reads
    case typeDevWrites: curr = devStat.writes
    case typeDevReadBytes: curr = devStat.readSectors * 512 // 1 sector is 512 bytes
    case typeDevWriteBytes: curr = devStat.writeSectors * 512
    }

    prevDevStats[typ][dev] = curr
    if !prevOk {return float64(0.0)}

    return float64(curr - prev)

}

func GetDevReads(key string) interface{} {
    return getDevStat(key, typeDevReads, false)
}

func GetDevWrites(key string) interface{} {
    return getDevStat(key, typeDevWrites, false)
}

func GetDevReadBytes(key string) interface{} {
    return getDevStat(key, typeDevReadBytes, false)
}

func GetDevWriteBytes(key string) interface{} {
    return getDevStat(key, typeDevWriteBytes, false)
}


// STATFS ---

const (
    typeDevUsage = iota
    typeDevSize
)

func getStatfs(key string, typ int, nested bool) interface{} {

    if !nested {
        parseDevStats()
        devMutex.RLock()
        defer devMutex.RUnlock()
    }

    if key == "*" {
        ret := make(map[string] float64)
        for dev := range parsedStatfs {
            out := getStatfs(dev, typ, true)
            if out == nil {continue}
            ret[dev] = out.(float64)
        }
        if len(ret) == 0 {return nil}
        return ret
    }

    dev := getDev(key)
    if dev == "" {return nil}

    statfs, statfsOk := parsedStatfs[dev]
    if !statfsOk {return nil}

    switch typ {
    case typeDevUsage: return (1.0 - float64(statfs.free) / float64(statfs.blocks)) * 100.0
    case typeDevSize: return float64(statfs.blocks * statfs.blocksize)
    }

    return nil

}

func GetDevUsage(key string) interface{} {
    return getStatfs(key, typeDevUsage, false)
}

func GetDevSize(key string) interface{} {
    return getStatfs(key, typeDevSize, false)
}

// IO USAGE
// https://serverfault.com/questions/862334/interpreting-read-write-and-total-io-time-in-proc-diskstats

var prevIoTicks = make(map[string] int64)
var lastIoTick  = make(map[string] time.Time)
func getDevIoUsage(key string, nested bool) interface{} {

    if !nested {
        parseDevStats()
        devMutex.RLock()
        defer devMutex.RUnlock()
    }

    if key == "*" {
        ret := make(map[string] float64)
        for dev := range parsedDevStats {
            out := getDevIoUsage(dev, true)
            if out == nil {continue}
            ret[dev] = out.(float64)
        }
        if len(ret) == 0 {return nil}
        return ret
    }

    dev := getDev(key)
    if dev == "" {return nil}

    now  := time.Now()
    last, lastOk := lastIoTick[dev]
    if !lastOk {
        last = systemStartTime
    }
    past := now.Sub(last) / time.Millisecond
    lastIoTick[dev] = now

    ds   := parsedDevStats[dev]
    prev := prevIoTicks[dev]
    prevIoTicks[dev] = ds.ioTicks

    return float64(ds.ioTicks - prev) / float64(past) * 100.0

}

func GetDevIoUsage(key string) interface{} {
    return getDevIoUsage(key, false)
}
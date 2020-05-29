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

// statvfs
// https://www.man7.org/linux/man-pages/man2/statvfs.2.html

// Use /proc/self/mountinfo to get mounted disks

// df uses statvfs syscall to get disk informations and
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
// statvfs with the mount point
// e.g., disk-usage(<devname/uuid/label/id/mount point>) => statvfs
// e.g., disk-io-usage... => /sys/block

import (
    "fmt"
    "path/filepath"
    "strings"
    "sync/atomic"
    "time"

    . "github.com/hjjg200/go-act"
    "github.com/hjjg200/go-together"
)

type devStatStruct struct { // /sys/dev/block/[major:minor]/stat
    reads        uint64
    readSectors  uint64 // 1 sector is 512 bytes
    readTicks    uint64 // millisecond
    writes       uint64
    writeSectors uint64
    writeTicks   uint64
    ioTicks      uint64 // millisecond
                        // readTicks + writeTicks != ioTicks
}

type statvfsStruct struct { // c statvfs
    blocksize uint64
    blocks    uint64
    free      uint64
}

type mountInfoStruct struct { // /proc/self/mountinfo
    mountId    int // 1
    parentId   int // 2
    majorMinor string // 3
    root       string // 4
    target     string // 5
    fsType     string // 9
    source     string // 10
}

const devParseMinimumWait = time.Millisecond * 10
var devRail = together.NewRailSwitch()
var devParsed int32

var devHierarchy map[string] []string
var devBlocks map[string] uint64
var devAliases map[string] []string
var devTargets map[string] []string
var devMajorMinors map[string] string

var parsedDevStats   map[string] devStatStruct
var parsedStatvfs    map[string] statvfsStruct
var parsedMountInfos []mountInfoStruct

func parseDevStats() {

    devRail.Queue(railWrite, 1)
    defer devRail.Proceed(railWrite)

    if atomic.CompareAndSwapInt32(&devParsed, 0, 1) {

        // Prepare
        parseDevHierarchy()
        parsedStatvfs = make(map[string] statvfsStruct)

        // Statvfs
        for dev, targets := range devTargets {

            var ok bool
            var statvfs Statvfs_t
            for _, target := range targets {
                var buf Statvfs_t
                err := Statvfs(target, &buf)
                if err == nil {
                    statvfs = buf
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
            if ^statvfs.Frsize == 0 || statvfs.Frsize == 0 {
                if ^statvfs.Bsize == 0 {continue}
                blocksize = uint64(statvfs.Bsize)
            } else {
                blocksize = uint64(statvfs.Frsize)
            }

            // Assign
            parsedStatvfs[dev] = statvfsStruct{
                blocksize: blocksize,
                blocks: statvfs.Blocks,
                free: statvfs.Bfree,
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
                reads, readMerges, readSectors, readTicks uint64
                writes, writeMerges, writeSectors, writeTicks uint64
                inFlight, ioTicks, timeInQueue uint64
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

func parseDevHierarchy() {

    // Aliases
    devAliases   = make(map[string] []string)
    byDirs, err := filepath.Glob("/dev/disk/by-*")
    if err == nil {
        for _, byDir := range byDirs {
            entries, err := filepath.Glob(byDir + "/*")
            if err != nil {continue}

            for _, entry := range entries {
                resolved, err := filepath.EvalSymlinks(entry)
                if err != nil {continue}

                dev := filepath.Base(resolved)
                if _, ok := devAliases[dev]; !ok {
                    devAliases[dev] = []string{}
                }
                devAliases[dev] = append(
                    devAliases[dev],
                    entry, filepath.Base(entry),
                )
            }
        }
    }

    // Mountinfo
    // https://github.com/coreutils/gnulib/blob/a2080f6506701d8d9ca5111d628607a6a8013f61/lib/mountlist.c#L469
    parsedMountInfos  = make([]mountInfoStruct, 0)
    procmi, err      := readFile("/proc/self/mountinfo")
    if err != nil {
        panic(err) // Unexpected
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
        Try(next("", "- ",  nil)) // ignore optional fields
        Try(next("%s", " ", &mi.fsType))
        Try(next("%s", " ", &mi.source))

        parsedMountInfos = append(parsedMountInfos, mi)

    }()}

    // Device hierarchy
    partitions, err := readFile("/proc/partitions")
    if err != nil {
        panic(err)
    }

    devHierarchy   = make(map[string] []string)
    devBlocks      = make(map[string] uint64)
    devTargets     = make(map[string] []string)
    devMajorMinors = make(map[string] string)
    for _, line := range strings.Split(partitions, "\n") {

        var (
            major, minor int
            blocks uint64
            name string
        )

        n, err := fmt.Sscanf(line, "%d %d %d %s", &major, &minor, &blocks, &name)
        if err != nil || n != 4 {
            continue
        }

        if name[:2] == "md"   {continue}
        if name[:4] == "loop" {continue}

        devHierarchy[name]   = make([]string, 0)
        devBlocks[name]      = uint64(blocks)
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
            if dev == dev2 {continue}

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

    // dev name & dev full name
    // e.g., xvda or /dev/xvda
    for _, dev := range allDevs {
        if dev == key || key == "/dev/" + dev {return dev}
    }

    // /dev/disk/by-* entries
    // e.g., cloudimg-rootfs or /dev/disk/by-label/cloudimg-rootfs
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

    // mount points
    // e.g., / or /var/telescribe
    for dev, targets := range devTargets {
        for _, target := range targets {
            if target == key {
                return dev
            }
        }
    }

    return ""

}

// STAT ---

const (
    typeDevReads = iota
    typeDevWrites
    typeDevReadBytes
    typeDevWriteBytes
)

var prevDevStats = make(map[int] map[string] uint64)
func getDevStat(key string, typ int) (float64, error) {

    devRail.Queue(railRead, 1)
    defer devRail.Proceed(railRead)

    dev := getDev(key)
    if dev == "" {return 0.0, fmt.Errorf("Not found")}

    if _, ok := prevDevStats[typ]; !ok {
        prevDevStats[typ] = make(map[string] uint64)
    }

    var curr uint64
    prev, prevOk    := prevDevStats[typ][dev]
    devStat, statOk := parsedDevStats[dev]
    if !statOk {return 0.0, fmt.Errorf("Stat not found")}

    switch typ {
    case typeDevReads: curr = devStat.reads
    case typeDevWrites: curr = devStat.writes
    case typeDevReadBytes: curr = devStat.readSectors * 512 // 1 sector is 512 bytes
    case typeDevWriteBytes: curr = devStat.writeSectors * 512
    }

    prevDevStats[typ][dev] = curr
    // Ignore uninitialized devices
    if !prevOk {return float64(0.0), fmt.Errorf("Not initialized")}

    return float64(curr - prev), nil

}

func GetDevReads(key string) (float64, error) {
    parseDevStats()
    return getDevStat(key, typeDevReads)
}

func GetDevWrites(key string) (float64, error) {
    parseDevStats()
    return getDevStat(key, typeDevWrites)
}

func GetDevReadBytes(key string) (float64, error) {
    parseDevStats()
    return getDevStat(key, typeDevReadBytes)
}

func GetDevWriteBytes(key string) (float64, error) {
    parseDevStats()
    return getDevStat(key, typeDevWriteBytes)
}

// Multiple

func getDevsStat(typ int) map[string] float64 {

    parseDevStats()
    devRail.Queue(railRead, 1)
    defer devRail.Proceed(railRead)

    ret := make(map[string] float64)
    for dev := range parsedDevStats {
        out, err := getDevStat(dev, typ)
        if err != nil {continue}
        ret[dev] = out
    }
    return ret

}

func GetDevsReads() map[string] float64 {
    return getDevsStat(typeDevReads)
}

func GetDevsWrites() map[string] float64 {
    return getDevsStat(typeDevWrites)
}

func GetDevsReadBytes() map[string] float64 {
    return getDevsStat(typeDevReadBytes)
}

func GetDevsWriteBytes() map[string] float64 {
    return getDevsStat(typeDevWriteBytes)
}


// STATVFS ---

const (
    typeDevUsage = iota
    // typeDevSize
)

func getStatvfs(key string, typ int) (float64, error) {

    devRail.Queue(railRead, 1)
    defer devRail.Proceed(railRead)

    dev := getDev(key)
    if dev == "" {return 0.0, fmt.Errorf("Not found")}

    statvfs, statvfsOk := parsedStatvfs[dev]
    if !statvfsOk {return 0.0, fmt.Errorf("Statvfs not found")}

    switch typ {
    case typeDevUsage: return (1.0 - float64(statvfs.free) / float64(statvfs.blocks)) * 100.0, nil
    // case typeDevSize: return float64(statvfs.blocks * statvfs.blocksize)
    }

    return 0.0, fmt.Errorf("Wrong type")

}

func GetDevUsage(key string) (float64, error) {
    parseDevStats()
    return getStatvfs(key, typeDevUsage)
}

func GetDevsUsage() map[string] float64 {

    parseDevStats()

    ret := make(map[string] float64)
    for dev := range parsedStatvfs {
        usage, _ := getStatvfs(dev, typeDevUsage)
        ret[dev] = usage
    }
    return ret

}

// BLOCKS ---

// /proc/partitions
//    Contains the major and minor numbers of each partition as well
//    as the number of 1024-byte blocks and the partition name.

func getDevSize(key string) (float64, bool) {
    
    devRail.Queue(railRead, 1)
    defer devRail.Proceed(railRead)

    dev := getDev(key)
    if dev == "" {return 0.0, false}

    blocks, ok := devBlocks[dev]
    return float64(blocks) * 1024.0 / 1000.0, ok // convert to 1024-byte block to SI kilobyte

}

func GetDevSize(key string) (float64, error) {

    parseDevStats()

    kb, ok := getDevSize(key)
    if !ok {return 0.0, fmt.Errorf("Not found")}
    return kb, nil

}

func GetDevsSize() map[string] float64 {

    parseDevStats()

    ret := make(map[string] float64)
    for dev := range devBlocks {
        ret[dev], _ = getDevSize(dev)
    }
    return ret

}

// IO USAGE
// https://serverfault.com/questions/862334/interpreting-read-write-and-total-io-time-in-proc-diskstats

var prevIoTicks = make(map[string] uint64)
var lastIoTick  = make(map[string] time.Time)
func getDevIoUsage(key string) (float64, error) {

    devRail.Queue(railRead, 1)
    defer devRail.Proceed(railRead)

    dev := getDev(key)
    if dev == "" {return 0.0, fmt.Errorf("Not found")}

    now  := time.Now()
    last, lastOk := lastIoTick[dev]
    if !lastOk {
        last = systemStartTime
    }
    past := now.Sub(last) / time.Millisecond
    lastIoTick[dev] = now

    ds           := parsedDevStats[dev]
    prev, prevOk := prevIoTicks[dev]
    prevIoTicks[dev] = ds.ioTicks
    if !prevOk {
        // Ignore uninitialized devices
        return 0.0, fmt.Errorf("Not initialized")
    }

    return float64(ds.ioTicks - prev) / float64(past) * 100.0, nil

}

func GetDevIoUsage(key string) (float64, error) {
    parseDevStats()
    return getDevIoUsage(key)
}

func GetDevsIoUsage() map[string] float64 {

    parseDevStats()

    ret := make(map[string] float64)
    for dev := range parsedDevStats {
        out, _ := getDevIoUsage(dev)
        ret[dev] = out
    }
    return ret

}
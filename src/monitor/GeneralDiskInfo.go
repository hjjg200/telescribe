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

import (
    "fmt"
    "path"
    "strings"
    "sync/atomic"
    "time"
)

type diskStat struct {
    filesystem string
    mount string
    blocks int64 // 1024-block
    used int64 // 1024-block
    reads int64
    readSectors int64 // 1 sector is 512 bytes
    readTicks int64
    writes int64
    writeSectors int64
    writeTicks int64
}

var diskHierarchy = make(map[string] []string)
var parsedDiskStats map[string] diskStat
const diskStatParsingMinimumWait = time.Millisecond * 10
var diskStatParsingWait int32

func init() {
    parseDiskHierarchy()
    parseDiskStats()

    // Initialize
    GetDiskWrites()
    GetDiskReads()
    GetDiskReadBytes()
    GetDiskWriteBytes()
    for _, ds := range parsedDiskStats {
        GetMountWrites(ds.mount)
        GetMountReads(ds.mount)
        GetMountWriteBytes(ds.mount)
        GetMountReadBytes(ds.mount)
    }
}

func parseDiskHierarchy() {

    pt, err := readFile("/proc/partitions")
    if err != nil {
        emitError(err)
    }

    for _, line := range strings.Split(pt, "\n") {
        var (
            major, minor, blocks int64
            name string
        )
        n, err := fmt.Sscanf(line, "%d %d %d %s", &major, &minor, &blocks, &name)
        if err != nil || n != 4 {
            continue
        }

        diskHierarchy[name] = make([]string, 0)
    }

    // Check partitions
    PartitionParseLoop:
    for name := range diskHierarchy {
        for name2 := range diskHierarchy {
            if name == name2 { continue }

            if strings.HasPrefix(name, name2) { // name is partition
                delete(diskHierarchy, name)
                // Add partition
                diskHierarchy[name2] = append(diskHierarchy[name2], name)
                continue PartitionParseLoop
            }
        }
    }

}

func getParentDisk(part string) string {
    for disk, parts := range diskHierarchy {
        for _, part2 := range parts {
            if part == part2 {
                return disk
            }
        }
    }
    return ""
}

func parseDiskStats() {

    if(atomic.CompareAndSwapInt32(&diskStatParsingWait, 0, 1) == false) {
        // Already running
        return
    }

    // Parse
    pds := make(map[string] diskStat)

    // 
    df, err := commandOutput("df -kP | grep ^/dev/")
    if err != nil {
        emitError(err)
    }

    for _, line := range strings.Split(df, "\n") {
        var (
            filesystem string
            blocks int64
            used int64
            available int64
            capacity string
            mount string
        )

        n, err := fmt.Sscanf(line, "%s %d %d %d %s %s", &filesystem, &blocks, &used, &available, &capacity, &mount)
        if err != nil || n != 6 {
            continue
        }

        name := path.Base(filesystem)
        pds[name] = diskStat{
            filesystem: filesystem,
            mount: mount,
            blocks: blocks,
            used: used,
        }
    }

    // Per device
    for name, ds := range pds {
        var (
            reads, readMerges, readSectors, readTicks int64
            writes, writeMerges, writeSectors, writeTicks int64
            inFlight, ioTicks, timeInQueue int64
        )
        disk := getParentDisk(name)
        st, err := readFile(fmt.Sprintf("/sys/block/%s/%s/stat", disk, name))
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

        ds.reads = reads
        ds.readSectors = readSectors
        ds.readTicks = readTicks
        ds.writes = writes
        ds.writeSectors = writeSectors
        ds.writeTicks = writeTicks

        // Assign
        pds[name] = ds
    }

    parsedDiskStats = pds

    time.Sleep(diskStatParsingMinimumWait)
    atomic.StoreInt32(&diskStatParsingWait, 0)

}

//
// Requests
//
var prevDiskWrites = make(map[string] int64)
func GetDiskWrites() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name := range pds {
        prev, _ := prevDiskWrites[name]
        ret[name] = float64(pds[name].writes - prev)
        prevDiskWrites[name] = pds[name].writes
    }
    return ret
}

var prevMountWrites = make(map[string] int64)
func GetMountWrites(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for name, ds := range pds {
        if ds.mount == m {
            prev, _ := prevMountWrites[m]
            prevMountWrites[m] = pds[name].writes
            return float64(pds[name].writes - prev), nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

var prevDiskReads = make(map[string] int64)
func GetDiskReads() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name := range pds {
        prev, _ := prevDiskReads[name]
        ret[name] = float64(pds[name].reads - prev)
        prevDiskReads[name] = pds[name].reads
    }
    return ret
}

var prevMountReads = make(map[string] int64)
func GetMountReads(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for name, ds := range pds {
        if ds.mount == m {
            prev, _ := prevMountReads[m]
            prevMountReads[m] = pds[name].reads
            return float64(pds[name].reads - prev), nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

//
// Bytes
//
var prevDiskWriteBytes = make(map[string] int64)
func GetDiskWriteBytes() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name := range pds {
        prev, _ := prevDiskWriteBytes[name]
        ret[name] = float64(pds[name].writeSectors - prev) * 512.0 // 1 secotr = 512 bytes
        prevDiskWriteBytes[name] = pds[name].writeSectors
    }
    return ret
}

var prevMountWriteBytes = make(map[string] int64)
func GetMountWriteBytes(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for name, ds := range pds {
        if ds.mount == m {
            prev, _ := prevMountWriteBytes[m]
            prevMountWriteBytes[m] = pds[name].writeSectors
            return float64(pds[name].writeSectors - prev) * 512.0, nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

var prevDiskReadBytes = make(map[string] int64)
func GetDiskReadBytes() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name := range pds {
        prev, _ := prevDiskReadBytes[name]
        ret[name] = float64(pds[name].readSectors - prev) * 512.0 // 1 secotr = 512 bytes
        prevDiskReadBytes[name] = pds[name].readSectors
    }
    return ret
}

var prevMountReadBytes = make(map[string] int64)
func GetMountReadBytes(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for name, ds := range pds {
        if ds.mount == m {
            prev, _ := prevMountReadBytes[m]
            prevMountReadBytes[m] = pds[name].readSectors
            return float64(pds[name].readSectors - prev) * 512.0, nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

func GetDiskUsage() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name, ds := range pds {
        blocks := float64(ds.blocks)
        used := float64(ds.used)
        ret[name] = used / blocks * 100.0
    }
    return ret
}

func GetMountUsage(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for _, ds := range pds {
        if ds.mount == m {
            blocks := float64(ds.blocks)
            used := float64(ds.used)
            return float64(used / blocks * 100.0), nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

func GetDiskSize() map[string] float64 {
    parseDiskStats()
    pds := parsedDiskStats
    ret := make(map[string] float64)

    for name, ds := range pds {
        ret[name] = float64(ds.blocks)
    }
    return ret
}

func GetMountSize(m string) (float64, error) {
    parseDiskStats()
    pds := parsedDiskStats

    for _, ds := range pds {
        if ds.mount == m {
            return float64(ds.blocks), nil
        }
    }
    return 0.0, fmt.Errorf("Not found")
}

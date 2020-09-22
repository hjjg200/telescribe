package main

import (
    "bytes"
    "encoding/binary"
    "encoding/gob"
    "compress/gzip"
    "fmt"
    "io"
    "io/ioutil"
    "math"
    "sync"
    "testing"
    "time"
    "runtime"
)

var t_randomMonitorDataMap MonitorDataMap

var t_mdencStartTime     time.Time
var t_mdencStartMemStats runtime.MemStats
func t_startMonitorDataEncodeBenchmark() {

    ensureTestMonitorDataMap()

    t_mdencStartTime = time.Now()
    runtime.ReadMemStats(&t_mdencStartMemStats)

}
func t_endMonitorDataEncodeBenchmark() {

    past := time.Now().Sub(t_mdencStartTime)

    memst := runtime.MemStats{}
    runtime.ReadMemStats(&memst)
    addedAlloc := memst.TotalAlloc - t_mdencStartMemStats.TotalAlloc
    addedSys := memst.Sys - t_mdencStartMemStats.Sys
    addedGC := memst.NumGC - t_mdencStartMemStats.NumGC

    fmt.Printf(
        "Elapsed: %v    Alloc: %d kB    Sys: %d kB    NumGC: %d\n",
        past, addedAlloc / 1024, addedSys / 1024, addedGC,
    )

}

func ensureTestMonitorDataMap() {

    metrics := 3
    eachLength := 7776000 // 3 months for 1 second interval

    if t_randomMonitorDataMap == nil {
        t_randomMonitorDataMap = make(MonitorDataMap)
        
        for i := 1; i <= metrics; i++ {
            key := fmt.Sprintf("key-%2d", i)
            md := make(MonitorData, eachLength)
            for j := range md {
                md[j] = MonitorDatum{
                    Timestamp: int64(j),
                    Value: 1.0,
                    Per: 1,
                }
            }

            t_randomMonitorDataMap[key] = md
        }
    }
    fmt.Printf("MonitorDataMap: %d metrics, each %d long\n", metrics, eachLength)
}

func TestMonitorDataSerialization(t *testing.T) {

    md := MonitorData{
        MonitorDatum{10, 2.0, 5},
        MonitorDatum{15, 0.5, 5},
        MonitorDatum{20, 2.1, 5},
        MonitorDatum{25, 1.5, 5},
        MonitorDatum{30, 2.0, 5},
        MonitorDatum{35, 2.1, 5},
        MonitorDatum{40, 2.0, 5},
        MonitorDatum{45, 2.5, 5},
    }

    t.Logf("MD 1: %v\n", md)

    serial := SerializeMonitorData(md)
    t.Logf("SERIAL: %v\n", serial)

    md2, err := DeserializeMonitorData(serial)
    if err != nil {
        t.Error(err)
    }

    t.Logf("MD 2: %v\n", md2)

}

// The first method that was used
// go test -run MonitorDataEncode_1 -timeout 10m -v

// MonitorDataMap: 5 metrics, each 7776000 long
// Elapsed: 10.750749041s      Alloc: 984422 kB
// MonitorDataMap: 15 metrics, each 7776000 long
// Elapsed: 4m26.569587818s    Alloc: 2953219 kB    Sys: 1961850 kB    NumGC: 3

func TestBenchmarkMonitorDataEncode_1(t *testing.T) {
    t_startMonitorDataEncodeBenchmark()

    for _, md := range t_randomMonitorDataMap {
        cmp := t_monitorDataEncode_1(md)
        io.Copy(ioutil.Discard, bytes.NewReader(cmp)) // first method does linear write
    }

    t_endMonitorDataEncodeBenchmark()
}
func t_monitorDataEncode_1(md MonitorData) []byte {

    buf := bytes.NewBuffer(nil)
    gzw := gzip.NewWriter(buf)
    enc := gob.NewEncoder(gzw)

    // |       entire content       |
    // | type | timestamps | values |

    if len(md) > 0 {

        enc.Encode("float64")

        timestamps := make([]int64, len(md))
        values     := make([]float64, len(md))
        pers       := make([]int32, len(md))
        for i := range md {
            timestamps[i] = md[i].Timestamp
            values[i]     = md[i].Value
            pers[i]       = md[i].Per
        }

        enc.Encode(timestamps)
        enc.Encode(values)
        enc.Encode(pers)

    } else {

        enc.Encode("nil")

    }

    // Return
    gzw.Close()
    return buf.Bytes()

}

// The second method
// go test -run MonitorDataEncode_2 -timeout 10m -v

// MonitorDataMap: 5 metrics, each 7776000 long
// Elapsed: 9.007371598s      Alloc: 764932 kB
// MonitorDataMap: 15 metrics, each 7776000 long
// Elapsed: 1m49.278758921s    Alloc: 2294781 kB    Sys: 2029500 kB    NumGC: 1
func TestBenchmarkMonitorDataEncode_2(t *testing.T) {
    t_startMonitorDataEncodeBenchmark()

    wg := sync.WaitGroup{}
    for _, md := range t_randomMonitorDataMap {
        wg.Add(1)
        cmp := t_monitorDataEncode_2(md)
        go func(buf []byte) {
            io.Copy(ioutil.Discard, bytes.NewReader(buf))
            wg.Done()
        }(cmp)
    }
    wg.Wait()

    t_endMonitorDataEncodeBenchmark()
}
func t_monitorDataEncode_2(md MonitorData) []byte {

    buf := bytes.NewBuffer(nil)
    gzw := gzip.NewWriter(buf)
    
    // Length
    binary.Write(gzw, binary.LittleEndian, int64(len(md)))

    // Write
    binary.Write(gzw, binary.LittleEndian, md)

    gzw.Close()
    return buf.Bytes()

}

// The third method
// go test -run MonitorDataEncode_3 -timeout 10m -v

// MonitorDataMap: 5 metrics, each 7776000 long
// Elapsed: 4.804808344s     Alloc: 307742 kB    Sys: 202950 kB
// MonitorDataMap: 15 metrics, each 7776000 long
// Elapsed: 14.183095068s    Alloc: 315690 kB    Sys: 405900 kB    NumGC: 0
func TestBenchmarkMonitorDataEncode_3(t *testing.T) {
    t_startMonitorDataEncodeBenchmark()

    maxLen := 0
    for _, md := range t_randomMonitorDataMap {
        if maxLen < len(md) {
            maxLen = len(md)
        }
    }
    maxRawSize := maxLen * 20 + 1024

    buf := bytes.NewBuffer(make([]byte, 0, maxRawSize))
    bMd := make([]byte, maxRawSize)
    for _, md := range t_randomMonitorDataMap {
        buf.Reset()
        t_monitorDataEncode_3(md, buf, bMd)
        _ = buf.Bytes()
    }

    t_endMonitorDataEncodeBenchmark()
}
func t_monitorDataEncode_3(
    md MonitorData,
    buf *bytes.Buffer,
    bMd []byte,
) {

    gzw := gzip.NewWriter(buf)
    
    // Binary
    order := binary.LittleEndian
    
    // Length
    bLength := make([]byte, 8) // uint64
    order.PutUint64(bLength, uint64(len(md)))

    gzw.Write(bLength)

    // MonitorData
    sz  := 20
    for i, datum := range md {
        base := i * sz
        order.PutUint64(bMd[base:base + 8],       uint64(datum.Timestamp))
        order.PutUint64(bMd[base + 8:base + 16],  math.Float64bits(datum.Value))
        order.PutUint32(bMd[base + 16:base + 20], uint32(datum.Per))
    }

    gzw.Write(bMd[:len(md) * sz])
    gzw.Close()

}


// The 4th method
// go test -run MonitorDataEncode_4 -timeout 10m -v
// Separate gzip method

// MonitorDataMap: 15 metrics, each 7776000 long
// Elapsed: 19.914684484s    Alloc: 35776 kB    Sys: 0 kB    NumGC: 0
func TestBenchmarkMonitorDataEncode_4(t *testing.T) {
    t_startMonitorDataEncodeBenchmark()

    for _, md := range t_randomMonitorDataMap {
        t_monitorDataEncode_4(md, ioutil.Discard)
    }

    t_endMonitorDataEncodeBenchmark()
}
var t_mdenc_4_maxMem = 4 * 1024 * 1024
var t_mdenc_4_buffer = bytes.NewBuffer(make([]byte, 0, t_mdenc_4_maxMem))
func t_monitorDataEncode_4(
    md MonitorData,
    w io.Writer,
) {

    mdSz  := 20
    lenMd := len(md)
    segNo := math.Ceil(
        float64(lenMd) * float64(mdSz) / float64(t_mdenc_4_maxMem),
    )
    lengths := evenlyDivide(lenMd, int(segNo))

    fmt.Println("segNo:", segNo)

    // Binary
    order := binary.LittleEndian

    // Size
    gzSz := 0

    accm := 0
    for _, length := range lengths {

        t_mdenc_4_buffer.Reset()
        gzw := gzip.NewWriter(t_mdenc_4_buffer)

        // Length
        bLength := make([]byte, 8) // uint64
        order.PutUint64(bLength, uint64(length))
        gzw.Write(bLength)

        bMd := make([]byte, mdSz)
        for _, datum := range md[accm:accm + length] {
            order.PutUint64(bMd[0:8],   uint64(datum.Timestamp))
            order.PutUint64(bMd[8:16],  math.Float64bits(datum.Value))
            order.PutUint32(bMd[16:20], uint32(datum.Per))
            gzw.Write(bMd)
        }

        accm += length

        // Gzip
        gzw.Close()
        gzLen := t_mdenc_4_buffer.Len()
        order.PutUint64(bLength, uint64(gzLen))

        w.Write(bLength)
        gzSz += t_mdenc_4_buffer.Len()
        w.Write(t_mdenc_4_buffer.Bytes())

    }

    fmt.Printf("Gzipped: %d kB\n", gzSz / 1024)

}

// ---

func BenchmarkGob_Write1000000uint64(b *testing.B) {
    for i := 0; i < b.N; i++ {
        slice := make([]uint64, 1000000)
        enc := gob.NewEncoder(ioutil.Discard)
        enc.Encode(slice)
    }
}

func BenchmarkBinary_Write1000000uint64(b *testing.B) {
    for i := 0; i < b.N; i++ {
        slice := make([]uint64, 1000000)
        for j := range slice {
            buf := make([]byte, 8)
            binary.LittleEndian.PutUint64(buf, slice[j])
            ioutil.Discard.Write(buf)
        }
    }
}

func BenchmarkCachedBytes_Write1000000uint64(b *testing.B) {
    cache := make([]byte, 8)
    for i := 0; i < b.N; i++ {
        slice := make([]uint64, 1000000)
        for _ = range slice {
            ioutil.Discard.Write(cache)
        }
    }
}

func evenlyDivide(a int, b int) []int {
    mod  := a % b
    each := a / b
    ret  := make([]int, b)
    for i := 0; i < b; i++ {
        ret[i] = each
        if mod > 0 {
            mod--
            ret[i] += 1
        }
    }
    return ret
}

func benchmarkThreadedBinaryWrite(b *testing.B, threads int, nums int) {

    for i := 0; i < b.N; i++ {
        wg := sync.WaitGroup{}
        wg.Add(threads)
        lengths := evenlyDivide(nums, threads)
        entire := make([]*bytes.Buffer, threads)

        for i, length := range lengths {
            go func(order, x int) {
                entire[order] = bytes.NewBuffer(nil)
                buf := make([]byte, 8)
                for j := 0; j < x; j++ {
                    binary.LittleEndian.PutUint64(buf, uint64(0))
                    entire[order].Write(buf)
                }

                wg.Done()
            }(i, length)
        }

        wg.Wait()
        for _, buffer := range entire {
            ioutil.Discard.Write(buffer.Bytes())
        }

    }

}

// 116640000 = 3 months long for 1 second and 15 metrics
// 11664000  = 116640000 / 10
// BenchmarkThreadedBinaryWrite_1_11664000                2         722586220 ns/op
// BenchmarkThreadedBinaryWrite_4_11664000                4         824950634 ns/op
// BenchmarkThreadedBinaryWrite_10_11664000               7         165045612 ns/op
// BenchmarkThreadedBinaryWrite_20_11664000               7         162970627 ns/op
// BenchmarkThreadedBinaryWrite_30_11664000               6         182694294 ns/op
func BenchmarkThreadedBinaryWrite_1_11664000(b *testing.B) {
    benchmarkThreadedBinaryWrite(b, 1, 11664000)
}

func BenchmarkThreadedBinaryWrite_4_11664000(b *testing.B) {
    benchmarkThreadedBinaryWrite(b, 4, 11664000)
}

func BenchmarkThreadedBinaryWrite_10_11664000(b *testing.B) {
    benchmarkThreadedBinaryWrite(b, 10, 11664000)
}

func BenchmarkThreadedBinaryWrite_20_11664000(b *testing.B) {
    benchmarkThreadedBinaryWrite(b, 20, 11664000)
}

func BenchmarkThreadedBinaryWrite_30_11664000(b *testing.B) {
    benchmarkThreadedBinaryWrite(b, 30, 11664000)
}
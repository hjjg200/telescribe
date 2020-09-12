package main

import (
    "bufio"
    "bytes"
    "compress/gzip"
    "encoding/binary"
    "encoding/gob"
    "fmt"
    "math"
    "./monitor"
    "./secret"

    . "github.com/hjjg200/go-act"
)

func init() {
    // Monitor error callback
    monitor.ErrorCallback = func(err error) {
        EventLogger.Warnln("monitor:", err)
    }
}

// STATUS ---

const (
    MonitorStatusNormal  = 0
    MonitorStatusWarning = 8
    MonitorStatusFatal   = 16
)

// DATA ---

type MonitorDataTableBox struct {
    Boundaries []byte
    DataMap    map[string/* monitorKey */] []byte
}

type MonitorDataMap map[string/* monitorKey */] MonitorData
type MonitorData []MonitorDatum
const MonitorDatumVersion uint8 = 1
const MointorDatumSize int = 20
type MonitorDatum struct {
    Timestamp int64 // timestamp is int64 as go uses int64 for unix timetamps
    Value float64
    Per int32
}

// Index ---
type MonitorDataIndex struct {
    UUID  string `json:"uuid"`
    Order uint64 `json:"order"`
    From  int64 `json:"from"`
    To    int64 `json:"to"`
    ptr   *MonitorData
}
type MonitorDataIndexes []MonitorDataIndex
type MonitorDataIndexesMap map[string/* monitorKey */] MonitorDataIndexes

func SliceAndIndexMonitorData(md MonitorData, sz uint64) MonitorDataIndexes {

    // Slices and indexes the given monitor data
    mdIndexes := make(MonitorDataIndexes, 0)
    left      := uint64(len(md))

    for i := uint64(0);; {

        if left == 0 {
            break
        }

        from   := i * sz
        length := sz
        if left < length {
            length = left
        }

        // Index
        uuidBytes := secret.RandomBytes(64)
        uuid      := fmt.Sprintf("%x", uuidBytes)
        data      := md[from:from + length]
        fromDatum := data[0]
        toDatum   := data[length - 1]
        
        mdIdx := MonitorDataIndex{
            UUID: uuid,
            Order: i,
            From: fromDatum.Timestamp,
            To: toDatum.Timestamp,
            ptr: &data,
        }
        mdIndexes = append(mdIndexes, mdIdx)

        // Post
        i++
        left -= length

    }

    return mdIndexes

}
func(mdIndexes MonitorDataIndexes) Append(rhs MonitorDataIndexes) MonitorDataIndexes {

    // Copy
    copied := make(MonitorDataIndexes, len(rhs))
    copy(copied, rhs)

    // If nil or zero
    // because of this, the function returns indexes, not utilizing reference
    if mdIndexes == nil || len(mdIndexes) == 0 {
        return copied
    }

    // Append the rhs indexes to the lhs indexes
    length   := len(mdIndexes)
    maxOrder := mdIndexes[length - 1].Order
    copied.orderOffset(maxOrder)

    return append(mdIndexes, copied...)

}
func(mdIndexes MonitorDataIndexes) orderOffset(offset uint64) {
    // Adds a defined value to all the orders of the indexes
    offset += 1
    for _, mdIdx := range mdIndexes {
        mdIdx.Order += offset
    }
}

// sort.Interface
func(md MonitorData) Len() int { return len(md) }
func(md MonitorData) Less(i, j int) bool { return md[i].Timestamp < md[j].Timestamp }
func(md MonitorData) Swap(i, j int) { md[i], md[j] = md[j], md[i] }
// For LTTB
func(datum MonitorDatum) X() float64 { return float64(datum.Timestamp) }
func(datum MonitorDatum) Y() float64 { return datum.Value }

// CONFIG ---

type MonitorConfig struct {
    Absolute     bool   `json:"absolute"`
    Alias        string `json:"alias"`
    Constant     bool   `json:"constant"`
    Format       string `json:"format"`
    FatalRange   Range  `json:"fatalRange"`
    WarningRange Range  `json:"warningRange"`
}
type MonitorConfigMap map[string/* monitorKey */] MonitorConfig

func(mCfg MonitorConfig) StatusOf(val float64) int {
    switch {
    case mCfg.FatalRange.Includes(val):
        return MonitorStatusFatal
    case mCfg.WarningRange.Includes(val):
        return MonitorStatusWarning
    }
    return MonitorStatusNormal
}

// KEY ---

type monitorKey struct {
    base, param, idx string
}
var parsedMonitorKeys = make(map[string] monitorKey)

func ParseMonitorKey(mKey string) (base, param, idx string) {
    if m, ok := parsedMonitorKeys[mKey]; ok {
        base, param, idx = m.base, m.param, m.idx
    } else {
        base, param, idx        = monitor.ParseWrapperKey(mKey)
        parsedMonitorKeys[mKey] = monitorKey{
            base, param, idx,
        }
    }
    return
}
func FormatMonitorKey(base, param, idx string) string {
    return monitor.FormatWrapperKey(base, param, idx)
}

// SERIALIZATION ---

func SerializeMonitorData(md MonitorData) []byte {

    // Type version check
    Assert(MonitorDatumVersion == 1, "MonitorDatum version must be 1")

    // Byte
    lenMd  := len(md)
    serial := make(
        []byte,
        1 + lenMd * MointorDatumSize, // version byte + data bytes
    )

    // Version byte
    serial[0] = byte(MonitorDatumVersion)

    // Data
    le   := binary.LittleEndian
    base := 1 // from 1st position
    for _, datum := range md {
        le.PutUint64(serial[base:base + 8],       uint64(datum.Timestamp))
        le.PutUint64(serial[base + 8:base + 16],  math.Float64bits(datum.Value))
        le.PutUint32(serial[base + 16:base + 20], uint32(datum.Per))

        base += MointorDatumSize
    }

    return serial

}

func DeserializeMonitorData(serial []byte) (md MonitorData, err error) {

    // Type version check
    Assert(MonitorDatumVersion == 1, "MonitorDatum version must be 1")

    defer Catch(&err) // recover from panic from this point

    // Version byte
    version := serial[0]
    Assert(uint8(version) == MonitorDatumVersion, "MonitorData version must match the current version")

    // 

    return nil, nil

}

// COMPRESSION ---

func CompressMonitorData(md MonitorData) (cmp []byte, err error) {

    defer Catch(&err)

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
    cmp = buf.Bytes()
    return

}

func DecompressMonitorData(cmp []byte) (md MonitorData, err error) {

    defer Catch(&err)

    rd       := bytes.NewReader(cmp)
    gzr, err := gzip.NewReader(bufio.NewReader(rd))
    Try(err)
    dec      := gob.NewDecoder(gzr)

    // Type
    var typ string
    dec.Decode(&typ)

    // Value
    switch typ {
    case "float64":

        timestamps := make([]int64, 0)
        values     := make([]float64, 0)
        pers       := make([]int32, 0)

        // Timestamp
        dec.Decode(&timestamps)
        dec.Decode(&values)
        dec.Decode(&pers)
        
        // Assign
        md = make(MonitorData, len(timestamps))
        for i := 0; i < len(timestamps); i++ {
            md[i] = MonitorDatum{
                Timestamp: timestamps[i],
                Value:     values[i],
                Per:       pers[i],
            }
        }

    case "nil":

        // Return empty
        md = make(MonitorData, 0)

    }

    return

}

// Implementation of Largest-Triangle-Three-Buckets down-sampling algorithm
// https://github.com/dgryski/go-lttb
func LttbMonitorData(md MonitorData, threshold int) MonitorData {

    if threshold >= len(md) || threshold == 0 {
        return md
    }

    sampled := make(MonitorData, 0, threshold)

    // Bucket size. Leave room for start and end data points
    every := float64(len(md) - 2) / float64(threshold - 2)

    sampled = append(sampled, md[0]) // Always add the first point

    bucketStart := 1
    bucketCenter := int(math.Floor(every)) + 1

    var a int

    for i := 0; i < threshold - 2; i++ {

        bucketEnd := int(math.Floor(float64(i + 2) * every)) + 1

        // Calculate point average for next bucket (containing c)
        avgRangeStart := bucketCenter
        avgRangeEnd := bucketEnd

        if avgRangeEnd >= len(md) {
            avgRangeEnd = len(md)
        }

        avgRangeLength := float64(avgRangeEnd - avgRangeStart)

        var avgX, avgY float64
        for ; avgRangeStart < avgRangeEnd; avgRangeStart++ {
            avgX += md[avgRangeStart].X()
            avgY += md[avgRangeStart].Y()
        }
        avgX /= avgRangeLength
        avgY /= avgRangeLength

        // Get the range for this bucket
        rangeOffs := bucketStart
        rangeTo := bucketCenter

        // Point a
        pointAX := md[a].X()
        pointAY := md[a].Y()

        maxArea := -1.0

        var nextA int
        for ; rangeOffs < rangeTo; rangeOffs++ {
            // Calculate triangle area over three buckets
            area := (pointAX - avgX) * (md[rangeOffs].Y() - pointAY) -
                (pointAX - md[rangeOffs].X()) * (avgY - pointAY)
            // We only care about the relative area here.
            // Calling math.Abs() is slower than squaring
            area *= area
            if area > maxArea {
                maxArea = area
                nextA = rangeOffs // Next a is this b
            }
        }

        sampled = append(sampled, md[nextA]) // Pick this point from the bucket
        a = nextA                               // This a is the next a (chosen b)

        bucketStart = bucketCenter
        bucketCenter = bucketEnd
    }

    sampled = append(sampled, md[len(md) - 1]) // Always add last

    return sampled
}
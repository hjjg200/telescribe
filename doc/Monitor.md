# Monitor

This documentation contains the specifications for monitor-related information.

## Key

|Go|Javascript|HTML|
|-|-|-|
|`mKey`|`monitorKey`|`monitor-key`|

A key can be divided into three parts: base, parameter, and index; `<base>(<parameter>)[<index>]`. Base represents what type of key it basically is; parameter acts as the parameter when the base key is mapped to a functional metrics; index acts as the index when the base metrics produce mapped values. When you would like to use `[]` or `()` as plain text, you must use double quotation marks to enclose them: `someKey["a[1]"]`.

|Key|Description|
|-|-|
|`cpu-count`|The count of CPUs|
|`cpu-usage`|The usage of the CPU in percentage|
|`memory-size`|The amount of the installed memory in kB|
|`memory-size-mb`|The amount of the installed memory in MB|
|`memory-size-gb`|The amount of the installed memory in GB|
|`memory-usage`|The usage of the memory in percentage|
|`swap-size`|The amount of the swap in kB|
|`swap-size-mb`|The amount of the swap in MB|
|`swap-size-gb`|The amount of the swap in GB|
|`swap-usage`|The usage of the swap in percentage|
|`load`|Load average for 1m, 5m, and 15m durations|
|`load[1m]`|Load average for 1m|
|`load[5m]`|Load average for 5m|
|`load[15m]`|Load average for 15m|
|`load-perCpu`|Load average divided by the count of CPUs for 1m, 5m, and 15m durations|
|`load-perCpu[1m]`|Load average per cpu for 1m|
|`load-perCpu[5m]`|Load average per cpu for 5m|
|`load-perCpu[15m]`|Load average per cpu for 15m|
|`disk-writes`|Write operation count for the entire disks|
|`disk-writes[<deviceName>]`|Write operation count for the specified disk|
|`mount-writes(<mountPoint>)`|Write operation count for the disk mounted at the mount point|
|`disk-reads`|Read operation count for the entire disks|
|`disk-reads[<deviceName>]`|Read operation count for the specified disk|
|`mount-reads(<mountPoint>)`|Read operation count for the disk mounted at the mount point|
|`disk-writeBytes`|Written bytes of the entire disks|
|`disk-writeBytes[<deviceName>]`|Written bytes of the specified disk|
|`mount-writeBytes(<mountPoint>)`|Written bytes of the disk mounted at the mount point|
|`disk-readBytes`|Read bytes of the entire disks|
|`disk-readBytes[<deviceName>]`|Read bytes of the specified disk|
|`mount-readBytes(<mountPoint>)`|Read bytes of the disk mounted at the mount point|
|`disk-usage`|The usage of the entire disks|
|`disk-usage[<deviceName>]`|The usage of the specified disk|
|`mount-usage(<mountPoint>)`|The usage of the disk mounted at the mount point|
|`disk-size`|The size of the entire disks in kB|
|`disk-size[<deviceName>]`|The size of the specified disk in kB|
|`mount-size(<mountPoint>)`|The size of the disk mounted at the mount point in kB|
|`disk-size-mb`|The size of the entire disks in MB|
|`disk-size-mb[<deviceName>]`|The size of the specified disk in MB|
|`mount-size-mb(<mountPoint>)`|The size of the disk mounted at the mount point in MB|
|`disk-size-gb`|The size of the entire disks in GB|
|`disk-size-gb[<deviceName>]`|The size of the specified disk in GB|
|`mount-size-gb(<mountPoint>)`|The size of the disk mounted at the mount point in GB|
|`disk-size-tb`|The size of the entire disks in TB|
|`disk-size-tb[<deviceName>]`|The size of the specified disk in TB|
|`mount-size-tb(<mountPoint>)`|The size of the disk mounted at the mount point in TB|
|`disk-io-usage`|The IO usage of the entire disks in percentage|
|`disk-io-usage[<deviceName>]`|The IO usage of the specified disk in percentage|
|`mount-io-usage(<mountPoint>)`|The IO usage of the disk mounted at the mount point in percentage|
|`network-in`|Incoming bytes of the entire interfaces|
|`network-in[<interfaceName>]`|Incoming bytes of the specified interface|
|`network-inPackets`|Incoming packets of the entire interfaces|
|`network-inPackets[<interfaceName>]`|Incoming packets of the specified interface|
|`network-out`|Outgoing bytes of the entire interfaces|
|`network-out[<interfaceName>]`|Outgoing bytes of the specified interface|
|`network-outPackets`|Outgoing packets of the entire interfaces|
|`network-outPackets[<interfaceName>]`|Outgoing packets of the specified interface|
|`process-cpu-usage(<pid/comm/arg0>)`|CPU usage of the specified processes in whole|
|`process-memory-usage(<pid/comm/arg0>)`|Memory usage of the speicifed processes in whole|
|`process-swap-usage(<pid/comm/arg0>)`|Memory usage of the speicifed processes in whole|
|`process-read-bytes(<pid/comm/arg0>)`|Bytes read by the speicifed processes in whole|
|`process-write-bytes(<pid/comm/arg0>)`|Bytes written by the speicifed processes in whole|
|`command(<string>)`|The output of the command|

### CPU

***#** `cpu-count`*

CPU count is evaluated from `/proc/stat` by counting the lines that can be expressed as `/^cpu\d+`

***#** `cpu-usage`*

CPU usage is evaluated from `/proc/stat` by exmaining the ratio of non-idle(idle and iowait) cpu time since the last evaluation of CPU usage.

### Memory

***#** `memory-size`*

Memory size is evaluated from `/proc/memstat`'s MemTotal entry which is expressed in kB.

***#** `memory-usage`*

Memory usage is evaluated from `/proc/memstat` by examining the ratio of used memory(non-buffer, non-cached, and non-free) size.

***#** `swap-size`*

Swap size is evaluated from `/proc/memstat`'s SwapTotal entry which is expressed in kB.

***#** `swap-usage`*

Swap usage is evaluated from `/proc/memstat` by examining the ratio of used swap(non-cached and non-free) size.)


### Load Average

### Disk

### Network

### Process

### Command

## Config

|Go|Javascript|HTML|
|-|-|-|
|`mCfg`|`monitorConfig`|`monitor-config`|

The monitor configuration contains information of fatal and warning ranges of values.

|Item|Description|
|-|-|
|`absolute`|Boolean for whether the data is time-independent|
|`alias`|The name shown on the web instead of the key itself|
|`constant`|Boolean for whether it is considered hardly changing and thus not stored for graph plotting|
|`format`|The **Web.Format** in which values are expressed; the actual value is not affected by this|
|`fatalRange`|The **Util.Range** in which values are considered fatal|
|`warningRange`|The **Util.Range** in which values are considered warning|


## Config Map

|Go|Javascript|HTML|
|-|-|-|
|`mCfgMap`|`monitorConfigMap`|`monitor-config-map`|


## Status

|Go|Javascript|HTML|
|-|-|-|
|`mStat`|`monitorStatus`|`monitor-status`|

|Value|Status|
|-|-|
|0|Normal|
|8|Warning|
|16|Fatal|


## Timestamp

|Go|Javascript|HTML|
|-|-|-|
|`ts`|`timestamp`|`timestamp`|

A timestamp is defined as unix timestamp in seconds and expressed as `int64`.


## Boundaries

|Go|Javascript|HTML|
|-|-|-|
|`bds`|`boundaries`|`boundaries`|


## Value

|Go|Javascript|HTML|
|-|-|-|
|`val`|`value`|`value`|

A value is defined as floating point value and expressed as `float64`.


## Value Map

|Go|Javascript|HTML|
|-|-|-|
|`valMap`|`valueMap`|`value-map`|

A value map can be expressed as `map[string] float64` in go. The keys are **Monitor.Key** and the values are **Monitor.Value**.


## Per

|Go|Javascript|HTML|
|`per`|`per`|`per`|

A per value represents the duration, in seconds, for its relevant datum; for example, a cpu usage datum whose value is 50% and per is 60 seconds indicates the cpu was at 50% of usage for the last 60 seconds.

This concept was adopted as sometimes, percent basis monitor items cannot fully reflect the resource usage, since you can change the monitor interval at any time and the data shown on the web get decimated for performance reasons.


## Datum

|Go|Javascript|HTML|
|-|-|-|
|`mDatum`|`monitorDatum`|`monitor-datum`|

A datum is a set of **Monitor.Timestamp**, **Monitor.Value**, and **Monitor.Per**. Each represents a recorded value at a certain time.

```go
type MonitorDatum struct {
    Timestamp int64
    Value float64
    Per int
}
```


## Data

|Go|Javascript|HTML|
|-|-|-|
|`mData`|`monitorData`|`monitor-data`|

Data is an array of **Monitor.Datum**. Typically, it is sorted in ascending order for timestamps.


### Compressed

The compression for **Monitor.Data** is done in the following manner in gzip encoding; the entire content of compressed data is as follows:

|Order|Encoding|Type|Description|
|-|-|-|-|
|1|gob|`string`|`"float64"` or `"nil"`|
|2|gob|`[]int64`|An array of **Monitor.Timestamp**|
|3|gob|`[]float64`|An array of **Monitor.Value**|
|4|gob|`[]int`|An array of **Monitor.Per**|


## Data Map

|Go|Javascript|HTML|
|-|-|-|
|`mdMap`|`monitorDataMap`|`monitor-data-map`|

A data map is a map of **Monitor.Data** whose keys are, typically, **Monitor.Key**.


## Data Table Box

|Go|Javascript|HTML|
|-|-|-|
|`mdtBox`|`monitorData`|`monitor-key`|


### Boundaries Table

|Column|Description|
|`timestamp`||


### Monitor Data Table

|Column|Description|
|`timestamp`||
|`value`||
|`per`||
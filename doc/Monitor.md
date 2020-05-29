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
|`dev-reads(device/uuid/label/.../mount)`|Read operation count for the entire disks|
|`dev-writes`|Write operation count for the entire disks|
|`dev-readBytes`|Read bytes of the entire disks|
|`dev-writeBytes`|Written bytes of the entire disks|
|`dev-usage`|The usage of the entire disks|
|`dev-size`|The size of the entire disks in kB|
|`dev-size-mb`|The size of the entire disks in MB|
|`dev-size-gb`|The size of the entire disks in GB|
|`dev-size-tb`|The size of the entire disks in TB|
|`dev-io-usage`|The IO usage of the entire disks in percentage|
|`devs-reads[<deviceName>]`|Read operation count for the specified disk|
|`devs-writes[<deviceName>]`|Write operation count for the specified disk|
|`devs-readBytes[<deviceName>]`|Read bytes of the specified disk|
|`devs-writeBytes[<deviceName>]`|Written bytes of the specified disk|
|`devs-usage[<deviceName>]`|The usage of the specified disk|
|`devs-size[<deviceName>]`|The size of the specified disk in kB|
|`devs-size-mb[<deviceName>]`|The size of the specified disk in MB|
|`devs-size-gb[<deviceName>]`|The size of the specified disk in GB|
|`devs-size-tb[<deviceName>]`|The size of the specified disk in TB|
|`devs-io-usage[<deviceName>]`|The IO usage of the specified disk in percentage|
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

This is **indexed metrics** and its indexes are:

* `[0]`: The usage of the entire CPUs.
* `[<n>]`: The usage of the nth CPU.


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

***#** `load`*

Load average is evaluated from `/proc/loadavg` by taking its first three elements.

This is **indexed metrics** and its indexes are:

* `[1m]`: Load average for 1 minute
* `[5m]`: Load average for 5 minutes
* `[15m]`: Load average for 15 minutes

***#** `load-perCpu`*

Load average per cpu is evaluated by dividing the load average by the number of CPUs as specified by `cpu-count`.

This is **indexed metrics** and its indexes are the same as `load`


### Devices

***#** `dev-reads`*

Device reads is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This metrics **accept a parameter** and it can be:

* **Device name:** `xvda1`, `sdb`
* **Device full name:** `/dev/xvda`, `/dev/sda`
* **Device alias:** `/dev/disk/by-*/<alias>` or just `<alias>`
  * **Device UUID:** `556ce1a0-...` or `/dev/disk/by-uuid/556ce1a0-...`
  * **Device label:** `cloudimg-rootfs` or `/dev/disk/by-label/cloudimg-rootfs...`
* **Device mount point:** `/`, `/var/telescribe`


***#** `dev-writes`*

Device writes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-readBytes`*

Device read bytes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-writeBytes`*

Device write bytes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-usage`*

Device usage is evaluated from the `statvfs` result: `(1 - (f_free / f_blocks)) * 100`.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-size`*

Device size is evaluated from `/proc/partitions` and it is converted into SI prefix `kB` from 1024-byte block unit.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-size-mb`*

Device size MB is device size in SI prefix `MB`(1e+6 bytes).

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-size-gb`*

Device size GB is device size in SI prefix `GB`(1e+9 bytes).

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-size-tb`*

Device size TB is device size in SI prefix `TB`(1e+12 bytes).

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `dev-io-usage`*

Device IO usage is evaluated from `/sys/dev/block/[major:minor]/stat` by examining the ratio of elapsed io ticks(milliseconds) during the period since the last evaluation.

This metrics **accept a parameter** and it is as defined under `dev-reads`.


***#** `devs-reads`*

Device reads is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This is **indexed metrics** and its indexes are:

* `[<device_name>]`: short device names such as `xvda`, `sdb`, and `sdb1`


***#** `devs-writes`*

Device writes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-readBytes`*

Device read bytes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-writeBytes`*

Device write bytes is evaluated from `/sys/dev/block/[major:minor]/stat` by subtracting the last reads count from the current one.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-usage`*

Device usage is evaluated from the `statvfs` result: `(1 - (f_free / f_blocks)) * 100`.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-size`*

Device size is evaluated from `/proc/partitions` and it is converted into SI prefix `kB` from 1024-byte block unit.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-size-mb`*

Device size MB is device size in SI prefix `MB`(1e+6 bytes).

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-size-gb`*

Device size GB is device size in SI prefix `GB`(1e+9 bytes).

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-size-tb`*

Device size TB is device size in SI prefix `TB`(1e+12 bytes).

This is **indexed metrics** and its indexes are the same as `devs-reads`.


***#** `devs-io-usage`*

Device IO usage is evaluated from `/sys/dev/block/[major:minor]/stat` by examining the ratio of elapsed io ticks(milliseconds) during the period since the last evaluation.

This is **indexed metrics** and its indexes are the same as `devs-reads`.


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
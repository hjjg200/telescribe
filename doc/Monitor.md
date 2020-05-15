# Monitor

This documentation contains the specifications for monitor-related information.

## Key

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
|`command(<string>)`|The output of the command|

## Config

The monitor configuration contains information of fatal and warning ranges of values.

|Item|Description|
|-|-|
|`range.fatal`|The **Util.Range** in which values are considered fatal|
|`range.warning`|The **Util.Range** in which values are considered warning|
|`format`|The **Web.Format** in which values are expressed; the actual value is not affected by this|
|`constant`|Boolean for whether it is considered hardly changing and thus not stored for graph plotting|

## Status

|Value|Status|
|-|-|
|`0`|Normal|
|`8`|Warning|
|`16`|Fatal|
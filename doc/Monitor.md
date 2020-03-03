# Monitor

This documentation contains the specifications for monitor-related information.

## Key

A key is mapped to its unique metric type. When it is written with `[]` or `()` its behavior slightly changes; in most cases, `[]` are used for indexing as in `load[1m]` and `disk-usage[xvda1]`; `()` are used as functional parameters as in `command(echo 1)`. You can use double quotation marks when you want to nest `[]` or `()` in others; e.g., `example["a[1]"]`.

|Key|Description|
|-|-|
|`cpu-count`|The count of CPUs|
|`cpu-usage`|The usage of the CPU in percentage|
|`memory-total`|The amount of the installed memory in kB|
|`memory-usage`|The usage of the memory in percentage|
|`swap-total`|The amount of the swap in kB|
|`swap-usage`|The usage of the swap in percentage|
|`load`|Load average for 1m, 5m, and 15m durations|
|`load[1m]`|Load average for 1m|
|`load[5m]`|Load average for 5m|
|`load[15m]`|Load average for 15m|
|`load-perCpu`|Load average divided by the count of CPUs for 1m, 5m, and 15m durations|
|`load-perCpu[1m]`|Load average per cpu for 1m|
|`load-perCpu[5m]`|Load average per cpu for 5m|
|`load-perCpu[15m]`|Load average per cpu for 15m|
|`disk-writes`|Write operation counts for the entire disks|
|`disk-writes[<deviceName>]`|Write operation count for the specified disk|
|`mount-writes(<mountPoint>)`|Write operation count for the disk mounted at the mount point|
|`disk-reads`|Read operation counts for the entire disks|
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
|`network-in`|Incoming bytes of the entire interfaces|
|`network-in[<interfaceName>]`|Incoming bytes of the specified interface|
|`network-inPackets`|Incoming packets of the entire interfaces|
|`network-inPackets[<interfaceName>]`|Incoming packets of the specified interface|
|`network-out`|Outgoing bytes of the entire interfaces|
|`network-out[<interfaceName>]`|Outgoing bytes of the specified interface|
|`network-outPackets`|Outgoing packets of the entire interfaces|
|`network-outPackets[<interfaceName>]`|Outgoing packets of the specified interface|
|`command(<string>)`|The output of the command|


## Role

A role is a compound of monitoring configurations which is used to configure clients.

|Item|Description|
|-|-|
|`monitorConfigMap`|The map that contains **Monitor.Config** objects; keys of the map are **Monitor.Key**|
|`monitorInterval`|How often the client sends its metrics; in seconds|

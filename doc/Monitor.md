# Monitor

This documentation contains the specifications for monitor-related information.

## Key

A key is mapped to its unique metric type. When it is written with `[]` or `()` its behavior slightly changes specified in the below table; in most cases, `[]` are used for indexing as in `load[1m]` and `disk-usage[xvda1]`; `()` are used as functional parameters as in `command(echo 1)`. You can use double quotation marks when you want to nest `[]` or `()` in others; e.g., `example["a[1]"]`.

|Key|Description|
|-|-|
|`cpu-count`|The count of CPUs|
|`cpu-usage`|The usage of the CPU in percentage|
|`load`|Load average for 1m, 5m, and 15m durations|
|`load[1m]`|Load average for 1m|
|`load[5m]`|Load average for 5m|
|`load[15m]`|Load average for 15m|
|`load-perCpu`|Load average divided by the count of CPUs for 1m, 5m, and 15m durations|
|`load-perCpu[1m]`|Load average per cpu for 1m|
|`load-perCpu[5m]`|Load average per cpu for 5m|
|`load-perCpu[15m]`|Load average per cpu for 15m|
|`memory-total`|The amount of the installed memory in kB|
|`memory-usage`|The usage of the memory in percentage|
|`swap-total`|The amount of the swap in kB|
|`swap-usage`|The usage of the swap in percentage|


## Role

A role is a compound of monitoring configurations which is used to configure clients.

|Item|Description|
|-|-|
|`monitorConfigMap`|The map that contains Monitor.Config objects; keys of the map are Monitor.Key|
|`monitorInterval`|How often the client sends its metrics; in seconds|

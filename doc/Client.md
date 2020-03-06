# Client

This documentaion contains the specifications for client-related elements.

## ID

A client ID is typically key to a **Client.Info** object in maps, which must be unique to its own client.


## Info

|Item|Description|
|-|-|
|`Host`|The address of the client|
|`Alias`|The alias of the client|
|`Role`|The role of the client; roles are separated by spaces; roles get overlapped in sequence|


## Role

A role is a compound of monitoring configurations which is used to configure clients.

|Item|Description|
|-|-|
|`MonitorConfigMap`|The map that contains **Monitor.Config** objects; keys of the map are **Monitor.Key**|
|`MonitorInterval`|How often the client sends its metrics; in seconds|


## Status

A status object typically has the most recent information about the client. This should not be confused with **Monitor.Status**.

|Item|Description|
|-|-|
|`Timestamp`|Latest timestamp|
|`Value`|Latest value|
|`Status`|Integer value as documented at **Monitor.Status**|
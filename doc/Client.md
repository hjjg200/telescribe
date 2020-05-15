# Client

This documentaion contains the specifications for client-related elements.

## ID

A client ID is typically key to a **Client.Info** object in maps, which must be unique to its own client.


## Info

|Item|Description|
|-|-|
|`Host`|The address of the client|
|`Alias`|The alias of the client|
|`Tags`|The rule tags for the client; tags are separated by whitespaces; relevant rules get overlapped in order|


## Rule

A rule is a compound of monitoring configurations which is used to configure clients.

|Item|Description|
|-|-|
|`MonitorConfigMap`|The map that contains **Monitor.Config** objects; keys of the map are **Monitor.Key**|
|`MonitorInterval`|How often the client sends its metrics; in seconds|


## ItemStatus

A status object typically has the most recent status information about the client.

|Item|Description|
|-|-|
|`Timestamp`|Latest timestamp|
|`Value`|Latest value|
|`Status`|Integer value as documented at **Monitor.Status**|
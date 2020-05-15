# Client

This documentaion contains the specifications for client-related elements.

## ID

|Go|Javascript|HTML|
|-|-|-|
|clId|clientId|client-id|

A client ID is typically key to a **Client.Info** object in maps, which must be unique to its own client.


## Info

|Go|Javascript|HTML|
|-|-|-|
|clInfo|clientInfo|client-info|

|Item|Description|
|-|-|
|`Host`|The address of the client|
|`Alias`|The alias of the client|
|`Tags`|The rule tags for the client; tags are separated by whitespaces; relevant rules get overlapped in order|


## InfoMap

|Go|Javascript|HTML|
|-|-|-|
|infoMap|infoMap|info-map|


## Rule

|Go|Javascript|HTML|
|-|-|-|
|clRule|clientRule|client-rule|

A rule is a compound of monitoring configurations which is used to configure clients.

|Item|Description|
|-|-|
|`MonitorConfigMap`|The map that contains **Monitor.Config** objects; keys of the map are **Monitor.Key**|
|`MonitorInterval`|How often the client sends its metrics; in seconds|


## ItemStatus

|Go|Javascript|HTML|
|-|-|-|
|itemStat|itemStatus|item-status|

A status object typically has the most recent status information about the client.

|Item|Description|
|-|-|
|`Timestamp`|Typically the most recent **Monitor.Timestamp**|
|`Value`|Typically the most recent **Monitor.Value**|
|`Status`|Integer value as documented at **Monitor.Status**|
|`Constant`|Whether it is constant as specified in the relevant **Monitor.Config**|


## ItemStatusMap

|Go|Javascript|HTML|
|-|-|-|
|itemStatMap|itemStatusMap|item-status-map|
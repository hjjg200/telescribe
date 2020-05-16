# Server

This documentation contains the specifications of elements of server.


## Server Instance

|Go|
|-|
|`srv`|

## Server Config

|Go|Javascript|
|-|-|
|`srvCfg`|`serverConfig`|

By default, `serverConfig.json` contains the configuration of a server.

|Item|Description|
|-|-|
|`authPrivateKeyPath`|The path, either relative or absoulte, to the private key file used to sign data|
|`clientConfigPath`|The path, either relative or absoulte, to the config file that contains the entire client configuration|
|`http.users`|An array of **HTTP.User** objects|
|`http.certFilePath`|The path, either relative or absolute, to the certificate file used for SSL|
|`http.keyFilePath`|The path, either relative or absolute, to the key file used for SSL|
|`monitor.dataStoreInterval`|How oftern the server flushes the in-memory monitor data to files; in minutes|
|`monitor.dataStoreDir`|The path, either relative or absolute, to the directory that contains the entire stored monitor data files|
|`monitor.maxDataLength`|How many records for each monitor data that the server stores|
|`monitor.gapThresholdTime`|The time length by which the server determines whether there is a gap in the monitor data; in minutes|
|`monitor.decimationThreshold`|The number of records for each monitor data that the server decimates down to in order to increase the performance of graph drawing|
|`monitor.decimationInterval`|How often the server prepares the decimated version of monitor data; in minutes|
|`web`|A **Web.Config** object|
|`network.bind`|To which address the server binds its main listener|
|`network.port`|To which port the server opens its main listener|
|`network.tickrate`|How often the server handles incoming connections; in Hz|
|`alarm.webhookUrl`|The url the server sends fatal alarms to|


## Client Config

|Go|Javascript|HTML|
|-|-|-|
|`clCfg`|`clientConfig`|`client-config`|

By default, `clientConfig.json` contains the client configurations.

|Item|Description|
|-|-|
|`infoMap`|A map of **Client.Info**; keys of the map are **Client.ID** of each client|
|`ruleMap`|A map of **Client.Rule** objects which contain monitoring configuration; keys of the map are rule names of each rule|


## Monitor Data Store

The server flushes its in-memory monitor data to files as specified at `monitor.dataStoreInterval` and `monitor.dataStoreDir` in its configuration. And when flushing the data, it writes and compresses the files in the following format:

|Order|Encoding|Content|
|-|-|-|
|1|gob|**Client.ID**|
|2|gob|**Monitor.Key**|
|3|gob|**Monitor.Data.Compressed**|

And the name of each stored file is lowercase representation of sha256 sum for the concatenation of **Client.ID** and **Monitor.Key**, in order to maintain the consistency in the name length and the uniqueness of names. Last but not least, the extension for the files is `.store`


## Webhook

|Go|
|-|
|`webhook`|
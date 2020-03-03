# Server

This documentation contains the specifications of elements of server.

## serverConfig.json

By default, `serverConfig.json` contains the configuration of a server.

|Item|Description|
|-|-|
|`authPrivateKeyPath`|The path, either relative or absoulte, to the private key file used to sign data|
|`clientConfigPath`|The path, either relative or absoulte, to the config file that contains the entire client configuration|
|`http.users`|The array of **HTTP.User** objects|
|`http.certFilePath`|The path, either relative or absolute, to the certificate file used for SSL|
|`http.keyFilePath`|The path, either relative or absolute, to the key file used for SSL|
|`monitor.dataCacheInterval`|How oftern the server flushes the in-memory monitor data to files; in minutes|
|`monitor.dataCacheDir`|The path, either relative or absolute, to the directory that contains the entire monitor data cache files|
|`monitor.maxDataLength`|How many records for each monitor data that the server stores|
|`monitor.gapThresholdTime`|The time length by which the server determines whether there is a gap in the monitor data; in minutes|
|`monitor.decimationThreshold`|The number of records for each monitor data that the server decimates down to in order to increase the performance of graph drawing|
|`monitor.decimationInterval`|How often the server prepares the decimated version of monitor data; in minutes|
|`web`|The **Web.Config** object|
|`network.bind`|To which address the server binds its main listener|
|`network.port`|To which port the server opens its main listener|
|`network.tickrate`|How often the server handles incoming connections; in Hz|
|`alarm.webhookUrl`|The url the server sends fatal alarms to|

## clientConfig.json

By default, `clientConfig.json` contains the client configurations.

|Item|Description|
|-|-|
|`clientMap`|The map of **Client.Info**; keys of the map are ids of each client|
|`roles`|The map of **Monitor.Role** objects which contain monitoring configuration; keys of the map are role names of each role|

## API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

### Specification

|Item|Details|
|-|-|
|Base URI|`/api/v1/`|

### monitorDataTableBox

#### URL

`monitorDataTableBox/<fullName>/<mdKey>`

#### Permission

`api/v1.<method>.monitorDataTableBox.<fullName>.<mdKey>`

#### GET

Provides the user with the content of the specified monitorDataTableBox in the CSV form.

#### DELETE

Immediately deletes the specified monitor data's cache.
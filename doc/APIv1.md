## API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

### Specification

|Item|Details|
|-|-|
|Base URI|`/api/v1/`|

### `monitorDataTableBox/<fullName>/<mdKey>`

#### GET

* Permission: `api/v1.get.monitorDataTableBox.<fullName>.<mdKey>`

Provides the user with the content of the specified monitorDataTableBox in the CSV form.

#### DELETE

* Permission: `api/v1.delete.monitorDataTableBox.<fullName>.<mdKey>`

Immediately deletes the specified monitor data's cache.
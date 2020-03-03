# API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

## clientIds

#### URL

`/api/v1/clientIds`

#### Permission

`api/v1.<method>.clientIds.<clientId>`

#### GET

**200:** Provides the user with a JSON object that contains client id array.

```text
{
    "clientIds": [
        "...",
        "..."
    ]
}
```

**403:** No permission


## clientInfo

#### URL

`/api/v1/clientInfo/<clientId>`

#### Permission

`api/v1.<method>.clientInfo.<clientId>`

#### GET

**200:** Provides the user with a JSON object that contains client info.

```text
{
    "clientInfo": {
        "id": ...,
        "host": ...,
        "alias": ...,
        "role": ...
    }
}
```

**403:** No permission


## clientConfig

#### URL

`/api/v1/clientConfig/<clientId>`

#### Permission

`api/v1.<method>.clientConfig.<clientId>`

#### GET

**200:** Provides the user with a JSON object that contains client configuration.

```text
{
    "clientConfig": {
        "monitorConfig": {
            "<mdKey>": {
                "fatalRange": ...,
                "warningRange": ...,
                "format": ...
            }
        }
    }
}
```

**403:** No permission


## monitorDataBoundaries

#### URL

`/api/v1/monitorDataBoundaries/<clientId>`

#### Permission

`api/v1.<method>.monitorDataBoundaries.<clientId>`

#### GET

**200:** Provides the user with a csv table that contains timestamp boundaries of a client.

```text
timestamp
...
...
```

**403:** No permission


## monitorDataTable

#### URL

`/api/v1/monitorDataTable/<clientId>/<mdKey>`

#### Permission

`api/v1.<method>.monitorDataTable.<clientId>.<mdKey>`

#### GET

**200:** Provides the user with a csv table that contains timestamps and values.

```text
timestamp,value
...,...
...,...
```

**400:** Not found

**403:** No permission

#### DELETE

**200:** Immediately deletes the specified monitor data's cache.

**400:** Not found

**403:** No permission

**500:** Internal error; most likely an I/O error


## options

#### URL

`/api/v1/options`

#### Permission

`api/v1.<method>.options`

#### GET

**200:** Provides the user with a JSON object that contains the options

```text
{
    "options": {
        ...
    }
}
```

**403:** No permission


## version

#### URL

`/api/v1/version`

#### Permission

`api/v1.get.version`

#### GET

**200:** JSON object that contains the version
```text
{
    "version": "telescribe-..."
}
```
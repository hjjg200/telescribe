# API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

## Specification

|Item|Details|
|-|-|
|Base URL|`/api/v1/`|

## clientMap

#### URL

`clientMap`

#### Permission

`api/v1.<method>.clientMap.<fullName>`

#### GET

**200:** Provides the user with a JSON object that contains client list.

```text
{
    "clientMap": {
        "<fullName>": {
            "latestMap": {
                "<mdKey>": {
                    "timestamp": ...,
                    "value": ...,
                    "status": ...
                }
            },
            "configMap": {
                "<mdKey>": {
                    "fatalRange": ...,
                    "warningRange": ...,
                    "format": ...
                }
            }
        }
    }
}
```

**403:** No permission

## monitorDataTableBox

#### URL

`monitorDataTableBox/<fullName>/<mdKey>`

#### Permission

`api/v1.<method>.monitorDataTableBox.<fullName>.<mdKey>`

#### GET

**200:** Provides the user with the content of the specified monitorDataTableBox in the CSV form. When the `mdKey` is `boundaries` the csv only contains the timestamp column and the timestamp boundaries in it.

_boundaries
```text
timestamp
...
...
```

&lt;mdKey&gt;
```
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
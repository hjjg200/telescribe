# API v1

This documentation explains the behavior of API that is used to communicate between a server and a web client.

## clientMap

#### URL

`/api/v1/clientMap`

#### Permission

`api/v1.get.clientMap.<Client.ID>`

#### GET

**200:** Provides the user with a JSON object that contains a **Client.ID** to **Client.Info** map. The user will only have access to the clients he or she has access to, as defined in their permissions.

```text
{
    "clientMap": {
        "<Client.ID>": {
            Client.Info
        }
    }
}
```

**403:** No permission


## clientRole

#### URL

`/api/v1/clientRole/<Client.ID>`

#### Permission

`api/v1.get.clientRole.<Client.ID>`

#### GET

**200:** Provides the user with a **Client.Role** object.

```text
{
    "clientRole": {
        Client.Role
    }
}
```

**403:** No permission


## clientStatus

#### URL

`/api/v1/clientStatus/<Client.ID>`

#### Permission

`api/v1.get.clientStatus.<Client.ID>.<Monitor.Key>`

#### GET

* **200:** Provides the user with a **Monitor.Key** to **Client.Status** object.

```text
{
    "clientStatus": {
        "<Monitor.Key>": {
            Client.Status
        }
    }
}
```

* **400:** Not found

* **403:** No permission


## monitorDataBoundaries

#### URL

`/api/v1/monitorDataBoundaries/<Client.ID>`

#### Permission

`api/v1.get.monitorDataBoundaries.<Client.ID>`

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

`/api/v1/monitorDataTable/<Client.ID>/<Monitor.Key>`

#### Permission

`api/v1.<method>.monitorDataTable.<Client.ID>.<Monitor.Key>`

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


## webConfig

#### URL

`/api/v1/webConfig`

#### Permission

`api/v1.get.webConfig`

#### GET

**200:** Provides the user with a **Web.Config** object

```text
{
    "webConfig": {
        Web.Config
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
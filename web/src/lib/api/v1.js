
const apiName = "api/v1";

class apiError {
  constructor(uri, status) {
    this.uri = uri;
    this.status = status;
  }
  toString() {
    return `${apiName}: ${this.uri} returned a non-200 status code: ${this.status}`;
  }
}

function formatURI(key, ...args) {
  let suffix = args.length ? `/${args.join("/")}` : "";
  return `/${apiName}/${key}${suffix}`;
}

async function apiFetch(method, typ, key, ...args) {
  let uri = formatURI(key, ...args);
  let rsp = await fetch(uri, {method});
  if(rsp.status !== 200) {
    throw new apiError(uri, rsp.status);
  }
  return typ ? await rsp[typ]() : undefined;
}

export default {

  keyClientMap: "clientMap",
  async getClientMap() {
    return await apiFetch(
      "GET", "json", this.keyClientMap
    );
  },

  keyClientRole: "clientRole",
  async getClientRole(clId) { 
    return await apiFetch(
      "GET", "json", this.keyClientRole, clId
    );
  },

  keyClientStatus: "clientStatus",
  async getClientStatus(clId) { 
    return await apiFetch(
      "GET", "json", this.keyClientStatus, clId
    );
  },

  keyMcfg: "monitorConfig",
  async getMonitorConfig(clId, mKey) {
    return await apiFetch(
      "GET", "text", this.keyMcfg, clId, mKey
    );
  },

  keyMdb: "monitorDataBoundaries",
  async getMonitorDataBoundaries(clId) {
    return await apiFetch(
      "GET", "text", this.keyMdb, clId
    );
  },

  keyMdt: "monitorDataTable",
  async getMonitorDataTable(clId, mKey) {
    return await apiFetch(
      "GET", "text", this.keyMdt, clId, mKey
    );
  },
  async deleteMonitorDataTable(clId, mKey) {
    return await apiFetch(
      "DELETE", undefined, this.keyMdt, clId, mKey
    );
  },

  keyWebCfg: "webConfig",
  async getWebConfig() {
    return await apiFetch(
      "GET", "json", this.keyWebCfg
    );
  },

  keyVersion: "version",
  async getVersion() {
    return await apiFetch(
      "GET", "json", this.keyVersion
    );
  }

}
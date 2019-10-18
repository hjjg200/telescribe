
// For cleaning up the types

// host + alias
fullName
uid


type Server struct {

    clientMonitorDataTableClusters map[string] MonitorDataTableCluster
}

type ServerConfig struct {}
type ClientConfig struct {
    Hosts map[string] map[string] string `json:"hosts"`
    Roles map[string] ClientRole `json:"roles"`
}
type ClientRole struct { // role
    MonitorConfigMap map[string] MonitorConfig `json:"monitorConfigs"`
    MonitorInterval int `json:"monitorInterval"`
}
type Client struct {

}

type MonitorDataTableBox struct { // mdtBox
    Boundaries []byte
    DataMap map[string] []byte
}

type MonitorConfig struct { // mCfg
    WarningRange Range
    FatalRange Range
    Format string
}

type MonitorDataMap map[string] MonitorData // mdMap
type MonitorData []MonitorDatum // md
type MonitorDatum struct { // datum
    Timestamp int64
    Value float64
}

type ChartOptions struct { // chOpt
    Durations []int `json:"durations"`
    FormatNumber string `json:"format.number"`
    FormatDateLong string `json:"format.date.long"`
    FormatDateShort string `json:"format.date.short"`
}
type WebAbstract struct { // webAbs
    ClientMap map[string] WebABSClient `json:"clientMap"`
}
type WebABSClient struct { // webAbsCl
    CsvCluster WebABSCsvCluster `json:"csvCluster"`
    LatestMap map[string] WebABSLatest `json:"latestMap"`
    MonitorConfigMap map[string] MonitorConfig `json:"monitorConfigMap"`
}
type WebABSCsvCluster struct { // webAbsCsvC
    Boundaries string `json:"boundaries"`
    DataMap map[string] string `json:"dataMap"`
}
type WebABSLatest struct {
    Timestamp int64
    Value float64
    Status int
}


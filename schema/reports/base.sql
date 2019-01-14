CREATE TABLE IF NOT EXISTS proton.base_report (
    Schema                          String
    , Status                        Int16
    , HostnameID                    UInt64
    , ServerNameID                  UInt64
    , ScriptNameID                  UInt64
    , RequestCount                  UInt32
    , RequestTimeTotal              Float32
    , DocumentSizeTotal             UInt32
    , MemoryPeakTotal               UInt32
    , MemoryFootprintTotal          UInt32
    , UtimeTotal                    Float32
    , StimeTotal                    Float32

    , RequestTimeMax                AggregateFunction(Max, Float32)
    , DocumentSizeMax               AggregateFunction(Max, UInt32)
    , MemoryPeakMax                 AggregateFunction(Max, UInt32)
    , MemoryFootprintMax            AggregateFunction(Max, UInt32)
    , UtimeTotalMax                 AggregateFunction(Max, Float32)
    , StimeTotalMax                 AggregateFunction(Max, Float32)

    , RequestTimeTotalQuantiles     AggregateFunction(quantiles(0.9, 0.95, 0.99), Float32)
    , DocumentSizeTotalQuantiles    AggregateFunction(quantiles(0.9, 0.95, 0.99), UInt32)
    , MemoryPeakTotalQuantiles      AggregateFunction(quantiles(0.9, 0.95, 0.99), UInt32)
    , MemoryFootprintTotalQuantiles AggregateFunction(quantiles(0.9, 0.95, 0.99), UInt32)
    , UtimeTotalQuantiles           AggregateFunction(quantiles(0.9, 0.95, 0.99), Float32)
    , StimeTotalQuantiles           AggregateFunction(quantiles(0.9, 0.95, 0.99), Float32)
    , Timestamp                     DateTime
) Engine SummingMergeTree
PARTITION BY toYYYYMM(Timestamp)
ORDER BY (
    Schema
    , Status
    , HostnameID
    , ServerNameID
    , ScriptNameID
    , Timestamp
);

CREATE MATERIALIZED VIEW proton.v_base_report TO proton.base_report AS
    SELECT
        Schema
        , Status
        , cityHash64(Hostname)   AS HostnameID
        , cityHash64(ServerName) AS ServerNameID
        , cityHash64(ScriptName) AS ScriptNameID
        , CAST(1 AS UInt32)      AS RequestCount
        , RequestTime            AS RequestTimeTotal
        , DocumentSize           AS DocumentSizeTotal
        , MemoryPeak             AS MemoryPeakTotal
        , MemoryFootprint        AS MemoryFootprintTotal
        , Utime                  AS UtimeTotal
        , Stime                  AS StimeTotal
        , arrayReduce('maxState', [RequestTime])     AS RequestTimeMax
        , arrayReduce('maxState', [DocumentSize])    AS DocumentSizeMax
        , arrayReduce('maxState', [MemoryPeak])      AS MemoryPeakMax
        , arrayReduce('maxState', [MemoryFootprint]) AS MemoryFootprintMax
        , arrayReduce('maxState', [Utime])           AS UtimeTotalMax
        , arrayReduce('maxState', [Stime])           AS StimeTotalMax
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [RequestTime])     AS RequestTimeTotalQuantiles
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [DocumentSize])    AS DocumentSizeTotalQuantiles
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [MemoryPeak])      AS MemoryPeakTotalQuantiles
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [MemoryFootprint]) AS MemoryFootprintTotalQuantiles
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [Utime])           AS UtimeTotalQuantiles
        , arrayReduce('quantilesState(0.90,0.95,0.99)', [Stime])           AS StimeTotalQuantiles
        , toStartOfMinute(Timestamp) AS Timestamp
        /* ^^^ YOU CAN CHANGE IT.
         * Example:
         *   toStartOfFiveMinute(Timestamp)                 - round up to 5 minutes
         *   toDateTime(intDiv(toUInt32(Timestamp), 5) * 5) - round up to 5 seconds
         */
    FROM proton.requests;
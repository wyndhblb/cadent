
# gRPC api

When firing up the API sections to things, you can specify a gRPC (http://www.grpc.io/) endpoint for a few functions.

The gRPC API share the Indexer/Metric drivers for the normal http API.

Since gRPC is all protobuf all the time.  There are several protobuf sections that you may need if you wish to develope
your own gRPC consumers.

All the relevant protobuf files ::

- Stats: [src/cadent/server/schemas/repr/repr.proto](../src/cadent/server/schemas/metrics/metrics.proto)
- Index/Find: [src/cadent/server/schemas/indexer/indexer.proto](../src/cadent/server/schemas/indexer/indexer.proto)
- Metrics: [src/cadent/server/schemas/metrics/metrics.proto](../src/cadent/server/schemas/metrics/metrics.proto)
- Series: [src/cadent/server/schemas/series/series.proto](../src/cadent/server/schemas/series/series.proto)
- ApiQuery: [src/cadent/server/schemas/api/api.proto](../src/cadent/server/schemas/api/api.proto)
- gRPC: [src/cadent/server/writers/rpc/pb/cadent.proto](../src/cadent/server/writers/rpc/pb/cadent.proto)


Some of the generated protobuf code is manipulated by hand (only the tags) to allow MsgPACK and easyjson to produce their own goodies
from the resulting generation.  Notes about the changes are in the files.


## Basic Endpoints

    // GetMetrics.
    //
    // Obtains Metrics from a metrics query.
    //
    rpc GetMetrics(api.MetricQuery) returns (stream metrics.RawRenderItem) {}

    // GetMetricsInCache.
    //
    // Obtains Metrics from a metrics query, but only if the metrics are in local cache.
    // Usefull for doing fanout queries where only one node may have the most recent data.
    // If multiple hosts return values for the same metric, it is up to the client to merge the streams.
    //
    rpc GetMetricsInCache(api.MetricQuery) returns (stream metrics.RawRenderItem) {}

    // Find.
    //
    // Find full Metrics from a indexer query.
    //
    rpc Find(api.IndexQuery) returns (stream indexer.MetricFindItem) {}

    // FindInCache.
    //
    // Find full metrics items that are in the local RAM index
    // Usefull for doing fanout queries where only one node may have the metric in the local ram index.
    // If multiple hosts return values for the same metric, it is up to the client to merge the streams.
    //
    rpc FindInCache(api.IndexQuery) returns (stream indexer.MetricFindItem) {}

    // List.
    //
    // return a big list of paths
    //
    rpc List(api.IndexQuery) returns (stream indexer.MetricFindItem) {}

    // PutRawMetric.
    //
    // Write a raw metric into the metric writer. This is a direct writer, it does not consistently hash.
    //
    rpc PutRawMetric(series.RawMetric) returns (NoResponse) {}

    // PutStatRepr.
    //
    // Write a stat repr metric into the metric writer. This is a direct writer, it does not consistently hash.
    //
    rpc PutStatRepr(repr.StatRepr) returns (NoResponse) {}
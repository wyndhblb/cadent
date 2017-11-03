
API Clustering
--------------

Finally, proper "clustering" is supported for the basic API calls that feed into the graphite-ish system

By Clustering, we simply mean the api queries are fanned out amongst all registered cadents, and merged.

This is important while using different Ram indexes for different writers (where a given writer index has an internal metric cache
and index cache).  It's more important in the `injector` scenario where partitions are solidly split between various nodes.

Clustering uses the internal gRPC servers for all communication (as it's much faster then any other http like system).
As a result `grpc_listen` is required in the API configs for clustering support.

There are 2 modes

- Static: just a static list of the gRPC hosts (see below)
- Gossip: if you are using the gossip config, things will auto join and discover themselves (gossip is preferred in real-life scenarios)

See the `gossip` section ![confgtoml](./configtoml.md) doc. 

Clustering will only be enabled if we detect we can actually use it, otherwise things default back to the local node.

## Supported API calls

    {root}/find
    {root}/rawrender
    {root}/render
    {root}/list

## New API calls: Cluster List

This will return a host keyed set of data for each query (i.e. `{ host1: data, host2: data}`)

    {root}/cluster/find
    {root}/cluster/list
    {root}/cluster/rawrender
    
## New API calls: Local

This will return JUST the data on the local node you query that is in the cache.

    {root}/local/find
    {root}/local/list
    {root}/local/rawrender
            
## Config
 
An example `api` config is presented below.  If you want cluster to work properly `advertise_host` must be set


    [{thing}.api]
        base_path = "/"
        listen = "0.0.0.0:8084"
        
        # required for clustering
        grpc_listen = "0.0.0.0:8892"

        # set this to let the RPC clients in cluster know which in thehost is the "local" node
        # so we can optimize local requests, cadent will attempt to figure this out, but as we all know
        # based on binds and hostnames of the external vs internal and multi-net interfaces setting it
        # can avoid confusion.  If it's not set things will simply loop back to the grpc interface (even if its
        # the same server)
        advertise_host = "localhost:8892"

        # one can use a fixed list of hosts if desired, it's better to let the gossip
        # configure hosts as they go in and out
        # cadent_hosts = ["localhost:8891", "localhost:8892"]

        # enable GZIP output compression:
        # for large request rates (100-200TPS), gzip can cause ALOT of GC, the default is OFF
        # if your volume is lowish you should enable this, probably should not enable if
        # you are using msgpack/protobufs as the format for returns
        enable_gzip = false

        #  TLS the server if desired
        # key="/path/to/server.key"
        # cert="/path/to/server.crt"

        use_metrics="cass-metrics"
        use_indexer="cass-indexer"

See the 2 config files in ![injectors](../configs/injector/) to be able to run 2 partitioned kafka consumers (since things
are on "localhost" each config has a different port number for all the things).


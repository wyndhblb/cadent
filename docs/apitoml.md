
# API Configs

There are "3" modes we have.

The HTTP api, TCP api, and "SOLO" mode http api.

The "api" is a close mimic of the [Indexer and Writer sections](./preregtoml.md) w/o all the other "writing" related options.  There are
two main sections "metrics" and "indexer" like the writers and a few extra sections for other things.  
See the [api section](api.md) and the [gPRC doc](grpcapi.md) for more information

The drivers for the API DO NOT use the writer objects, but instead use another one that's simply focused on reading data
 from the various sources.  
 
The WriteBack cache IS SHARED across both writers and the tcp/http api readers.

The "read cache" section is in alpha mode, and is required for the experimental websocket endpoint.  The read cache does not use the 
"series" compression, but the internal format so we can easily manipulate it with golang slice operations.

It is best to use the `use_metrics` + `use_indexer` settings instead of the full configs as you will take advantage of the
caches, connection pools, etc from the raw writers.

OpenTracing via Zipkin or LightStep for http/gRPC requests as well, the default is not to have any tracing.


## HTTP API + gRPC Config

An example config

    [graphite-proxy-map.accumulator.api]
        
        # Create a simple gRPC server (this is a very different server then the standard http one below)
        # It shares the metrics and indexer engines.
        grpc_listen = "0.0.0.0:9089"
         
        # what ip/port to bind on
        listen = "0.0.0.0:8083"
        
        # all requests will be prefixed w/ this path 
        base_path = "/graphite/"


        # NOTE: Alpha, may change
        # this is the read cache that will keep the latest goods in ram
        # for all metrics and points this will be he max we will ever store in RAM
        read_cache_total_bytes=16384000
        
        # how many bytes to keep per activated metric in the cacn
        read_cache_max_bytes_per_metric=16384
        
        # Tag this API.  This is used in the Discovery module and /info endpoints
        # as metadata there
        tags = "zone=local,dc=myland,mode=test"
        
        # enable GZIP output compression:
        # for large request rates (>100 TPS), gzip can cause ALOT of GC, the default is OFF
        # if your volume is lowish you can enable this, you should NOT enable if
        # you are using msgpack/protobufs as the format for returns
        enable_gzip = false
        
        
        # For discovery/info if the acctuall endpoint is NOT the same as the hostname being used
        # once can set the name and url here
        # these will default to the hostname, hostname based url if blank
        # advertise_name=""
        # advertise_url=""
        
        # by defaults the http log will log to the same log as the over all cadent
        # this is a standard "apache" like log file
        # change that using this (stdout, stderr, path/to/file)
        # log_file = "stdout"
        
        # you can references the metrics/indexer sections OR you can define an entire other 
        # if these are defined then the "full config" below is ignored"
        use_metrics="my-metrics-writer-name"
        use_indexer="my-indexer-writer-name"
        
        # OpenTracing for API calls
        [graphite-proxy-map.accumulator.api.tracing]
            
            # zipkin endpoint
            zipkin_url="http://localhost:9411/api/v1/spans"
            
            # -- OR -- (cannot have mulitple tracers)
            # for LightStep
            # lightstep_key="XXXXX"
            
            # -- OR --
            # or for Jaeger
            # jaeger_host = "localhost:6831"
            
            # name of the service that will be registered inside zipkin
            name="cadent-graphite-whisper"
            
            # batch size number of traces to send in a batch 
            # if things are busy, set this to a higher number 
            # only matters for zipkin clients
            batch_size=5
            
            # Number of samples per second (0 == all traces, not to be used in production)
            # only matters for zipkin clients
            samples_per_second=100.0
            
        
        # SNIP :: ------  OR -------- it's better to use the above formats, rather then repeating yourself
        
        
        
        # METRICS api
        # This should mirror the same options as the Writer Metrics section above (in so far as pointing
        # things to the proper keyspace/database/tables, etc.  Things like the writes/second and rollup stuff
        # is not relevent here.  
        [graphite-proxy-map.accumulator.api.metrics]
                    
            driver = "cassandra-log-triggered" # mysql, cassandra-log, cassandra-log-triggered, etc
            dsn = "cassandrahost,cassandraotherhost"
            cache = "my-writer-cache"
            [graphite-proxy-map.accumulator.api.metrics.options]
                
                # use this keyspace
                keyspace="metric"
                
                # metrics table name/prefix
                metrics_table="metrics"
                
                # fast start
                sniff=false
        
        ## INDEXER api
        # This should mirror the same options as the Writer Metrics section above (in so far as pointing
        # things to the proper keyspace/database/tables, etc.  Things like the writes/second
        # is not relevent here. 
        [graphite-proxy-map.accumulator.api.indexer]
            driver = "elasticsearch"
            dsn = "http://127.0.0.1:9200"
            
        [graphite-proxy-map.accumulator.api.indexer.options]

            # authentication
            user=""
            password=""
            
            # path index
            path_index="metric_path"
            
            # segment index
            segment_table="metric_segment"
            
            # tag index
            tag_index="metric_tag"
            
            
        ## DISCOVERY module
        # Currently in alpha mode, but when the API fires up, it will register itself w/ a discovery
        # objects, so far only Zookeeper is created.
        [graphite-proxy-map.accumulator.api.discover]
            driver="zookeeper"
            dsn="127.0.0.1:2181"
            
            [graphite-proxy-map.accumulator.api.discover.options]
            
            # if false, don't "register" ourselves, but we can use the modle for reading from the discovery
            # object
            register=true 
            
            # Root Zookeeper path for all cadent related ZK activities
            root_path="/cadent"
            
            # sub path for API realated Zookeeper data
            api_path="/apihosts"
            
            # what name to register ourselves under
            # the default is to use the HostName of the host we are on.
            # the full path in ZK for a registered object woud be
            #
            # {root_path}/{api_path}/{name}
            #
            name=""


## TCP API Config

In beta, test away.  See the [tcp api section](tcpapi.md) for more information.
 

The config for this is very near the same the api section above WITHOUT the discover module.  It currently is assumed
that both TCP and HTTP api sections will be enabled.

    [graphite-proxy-map.accumulator.tcpapi]
        
        # what ip/port to bind on
        listen = "0.0.0.0:8084"
        
        # you can references the metrics/indexer sections OR you can define an entire other 
        # if these are defined then the "full config" below is ignored"
        use_metrics="my-metrics-writer-name"
        use_indexer="my-indexer-writer-name"
        
        # -------------- OR ----------------
        
        # METRICS api
        # This should mirror the same options as the Writer Metrics section above (in so far as pointing
        # things to the proper keyspace/database/tables, etc.  Things like the writes/second and rollup stuff
        # is not relevent here.  
        [graphite-proxy-map.accumulator.tcpapi.metrics]
                    
            driver = "cassandra-log-triggered" # mysql, cassandra-log, cassandra-log-triggered, etc
            dsn = "cassandrahost,cassandraotherhost"
            cache = "my-writer-cache"
            [graphite-proxy-map.accumulator.api.metrics.options]
                
                # use this keyspace
                keyspace="metric"
                
                # metrics table name/prefix
                metrics_table="metrics"
                
                # fast start
                sniff=false
        
        ## INDEXER api
        # This should mirror the same options as the Writer Metrics section above (in so far as pointing
        # things to the proper keyspace/database/tables, etc.  Things like the writes/second
        # is not relevent here. 
        [graphite-proxy-map.accumulator.tcpapi.indexer]
            driver = "elasticsearch"
            dsn = "http://127.0.0.1:9200"
            
        [graphite-proxy-map.accumulator.tcpapi.indexer.options]

            # authentication
            user=""
            password=""
            
            # path index
            path_index="metric_path"
            
            # segment index
            segment_table="metric_segment"
            
            # tag index
            tag_index="metric_tag"
            
 
## SOLO api mode (alpha, WIP)

WORK IN PROGRESS:

Several things need to be completed before this will move out of alpha, basically to use this effectively the discovery mechanism
needs to be tied in, in a preformat fashion (asking every standalone node for data is not efficent when there are 100s/1000s of nodes).


This is a mode in cadent that simply runs the API with out any writers.  Effectively it just connects to the backend data stores.
However, since there is alot of data in RAM from the actual writers,  it needs to be able to go to the writer hosts
and pull that cache data as well.  

In order to find those "writer" hosts, we are currently developing a "gossip" and/or "discovery" set of modules to facilitate
this.

The discovery module will also be important for `local` mode of cadent, where metrics are stored local to any node (writing to 
a levelDB backend).  In this mode, we will use the SOLO-LOCAL mode where it will proxy requests for specific hosts.

There is no current "TCP" solo mode api.  The TCPapi and or gRPC will likely be used for the local + cache pulling (depending on performance).

The basic (just read from a backend store) "SOLO" mode is currently available. Below is an example config.  

    ##
    ### just the render API and nothing else
    ##
    
    #############################
    ## logging
    #############################
    [log]
    # format = ""%{color}%{time:2006-01-02 15:04:05.000} [%{module}] (%{shortfile}) ▶ %{level:.4s} %{color:reset} %{message}""
    level = "DEBUG"
    file="stdout"
    
    
    #############################
    ## System things
    #############################
    [system]
    pid_file="/opt/cadent/api.pid"
    num_procs=4
    
    #############################
    ## CPU profile
    #############################
    [profile]
    ## Turn on CPU/Mem profiling
    ##  there will be a http server set up to yank pprof data on :6065
    enabled=true
    listen="0.0.0.0:6065"
    rate=100000
    block_profile=false # this is very expensive
    
    #############################
    ## Statsd
    #############################
    [statsd]
    server="statsdhost:8125"
    prefix="cadent"
    interval=1  # send to statd every second (buffered)
    
    # global statsd Sample Rates
    
    ## It's HIGHLY recommended you at least put a statsd_timer_sample_rate as we measure
    ## rates of very fast functions, and it can take _alot_ of CPU cycles just to do that
    ## unless your server not under much stress 1% is a good number
    ## `statsd_sample_rate` is for counters
    ## `statsd_timer_sample_rate` is for timers
    timer_sample_rate=0.01
    sample_rate=0.1
    
    #############################
    ## API
    #############################
    
    [api]
    base_path = "/graphite/"
    listen = "0.0.0.0:8086"
    
    # used to find the "writer" nodes as we'll need them to grab inflight/cached data
    # this will grab all the ones in the
    #
    # You can use the Seed or the Discover module (not both)  
    #
    # seed = "http://localhost:8083/graphite/info"
    
        [api.discover]
        driver = "zookeeper"
        dsn = "127.0.0.1:2181"
    
        [api.discover.options]
        register = false  # we don't want the solo mode to be registered as a writer host
    
        [api.metrics]
           driver = "cassandra-log-triggered"
           dsn = "localhost"
           [api.metrics.options]
               port=9042
               metrics_table="metric_series"
               keyspace="metric"
               path_table="path"
               segment_table="segment"
               writer_index=1
               sniff=false
    
    
        [api.indexer]
           driver = "cassandra"
           dsn = "localhost"
           [api.indexer.options]
               port=9042
               metrics_table="metric_series"
               keyspace="metric"
               path_table="path"
               segment_table="segment"

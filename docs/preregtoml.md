
# PreReg

Here one can define rules for moving inputs into different backends, as well as writers (indexers and metrics) and api/tcpapi hooks

This one has the most options and config parameters around, mostly due to the various different writer mechanisms. 

All the subsections below are contained in one file, but broken up here to keep things organized.  Please see the 
[config directory](../config/) for all sorts of varitions and settings.


## Shuttle mode

In this mode, there are no writers, simply a shuttling of a given line to another `server` section as defined in the [config.toml](./configtoml.md) section


    [graphite-proxy-map]
    
    # which [server.{name}] to hook into
    listen_server="graphite-proxy" 
    
    # failing any matches in the map below push lines to this [server.{name}]
    default_backend="graphite-nomatch"  
    
    
        # Substring type match: if the key has this text in it
        [[graphite-proxy-map.map]]
            substring=".marvel"  # ignore elastic search marvel stuff
            reject=true
        
        # Prefix match: if the key begins with this string
        [[graphite-proxy-map.map]]
            prefix="stats."  # farm all things begining with stats to the graphite-stats backend
            
            # push to this [server.{name}] section (must exist in the config.toml)
            backend="graphite-stats"  
                    
        # Regex match: if the key matches the regex
        [[graphite-proxy-map.map]]
            # match some kafka statsd we don't care about
            regex='''(kafka\.consumer\.FetchRequestAndResponseMetrics.*)|(.*ReplicaFetcherThread.*)|(kafka\.server\.FetcherLagMetrics\..*)|(kafka\.log\.Log\..*)|(kafka\.cluster\.Partition\..*)'''
            reject=true  
 

## Shuttle mode all bypass

In this mode, there are no writers, simply a shuttling of a given line to another `server` section as defined in the [config.toml](./configtoml.md) section.
But does not "do" any matching simply moves things from one frontend to another frontend


    [graphite-proxy-map]
    
    # which [server.{name}] to hook into
    listen_server="graphite-proxy" 
    
    # failing any matches in the map below push lines to this [server.{name}]
    default_backend="graphite-all"  
    
    
        # Do nothing, just push to default_backend
        [[graphite-proxy-map.map]]
            noop=true


## Accumulator mode

This mode "accumulates" metric of the same key  .. basically statsd.


### statsd example

    [statsd-regex-map]
    listen_server="statsd-proxy" # which listener to sit in front of  (must be in the main config)
    
    # This will never be used as the [...map] as the accumulator treats every incoming
    # metric.  It's here as cadent produce are error if not
    default_backend="statsd-proxy"  
    
        [statsd-regex-map.accumulator]
        backend = "graphite-relay"  # farm to graphite
        
        # we expect statsd like incoming lines
        input_format = "statsd"
        
        # we will output graphite lines from the resulting accumulation
        output_format = "graphite"
        
        # flush ever 10s (i.e. the reslution will be 10s .. minimum is 1s)
        accumulate_flush = "10s"  
        
        # accumulators can start on time % flush to get nice rounded numbers
        # for statsd -> graphite this is not nessesary, and if there is a farm of cadents doing this
        # it's best that they are randomized such that each cadent does not flush at the same time to
        # the same backend (i.e. many thousands of incoming lines all at the same time)
        random_ticker_start = true  
        
        # statsd options for the statsd accumulator
        options = [
                ["legacyNamespace", "true"],
                ["prefixGauge", "gauges"],
                ["prefixTimer", "timers"],
                ["prefixCounter", ""],  # legacy namespace
                ["globalPrefix", "stats"],
                ["globalSuffix", ""],
                ["percentThreshold", "0.90,0.95,0.99"]
        ]
    
        # Sub string type match
        [[statsd-regex-map.map]]
        substring=".marvel"  # ignore elastic search marvel stuff
        reject=true
    
        [[statsd-regex-map.map]]
        regex='''(kafka\.consumer\.FetchRequestAndResponseMetrics.*)|(.*ReplicaFetcherThread.*)|(kafka\.server\.FetcherLagMetrics\..*)|(kafka\.log\.Log\..*)|(kafka\.cluster\.Partition\..*)'''
        reject=true

        
## Accumulator + Writer mode

This mode "accumulates" metric of the same key  .. and then farms the results into writers.  Now things get a bit more 
 complicated in terms of configuration.  Each subsection is explained below.  For more information on writers please see
 the [writers doc](./writers.md)

For more information on `log`, `triggered`, and `bypass` modes see the [writers doc](./writers.md)


### graphite like example


    [graphite-proxy-map]
    listen_server="graphite-proxy" # which listener to sit in front of  (must be in the main config)
    default_backend="graphite-proxy"  # failing a match go here
    
        [graphite-proxy-map.accumulator]
        
        # BACKHOLE is a special "backend" that simply means the lines STOP here
        # you can specify another backend (good for replication) and things will both write 
        # to the writers below and shuttle off to another server set for whatever purpose
        
        backend = "BLACKHOLE"  
        input_format = "graphite"
        output_format = "graphite"
        
        # Flush every 5s
        accumulate_flush = "5s"
        
        # DO flush on time % flush == 0 for nice rounded times
        random_ticker_start = false
    
        # Aggregation Windows
        #
        # the Whisper format does the rollups internally, so there is not an extra write-back cache
        # for extra times here, the only one that is used is the `5s` one, however
        # the times + TTLs are here in order to CREATE the initial whisper file w/ the rollup times you wish.
        # please note that this will NOT change an existing whisper file if it is already on disk, this is just for 
        # new files
        #
        # For "triggered" rollup writers, the times specify which resolution to roll up into
        #
        # For non-triggered, non-whisper, and non-leveldb writers, each time gets its own cache  
        
        times = ["5s:1h", "1m:168h"]
    
        # BYPASS mode
        #
        # In ByPass mode, NOTHING is aggregated, and instead sent directly to the writers as is
        # This mode is false by default.  
        #`times` (above) are still important as, depending on the writer
        # mechanism, it will be needed for TTLs and rollups.
        
        bypass=false
        
        # Hash Mode
        #
        # the default hashing for unique ID generation is "fnv" if you desire one can use the farm hash
        #
        # hash_mode = "farm"
        
        # Tag mode
        #
        # the default tag mode is to use "metrics2" style tags and metatags, where the `tags` themselves
        # are derived from a fixed set of names, all other tags go to the metatags
        # metatags are NOT included in unique ID generation, if you want to ignore this default set the tag_mode=all
        # tag_mode = "all"
        
    
         # A write back caching object is required. (yes there can be multiple, explained below)
         [[graphite-proxy-map.accumulator.writer.caches]]
             ... see below ... 
    
         # Define the writer and associated options
         [graphite-proxy-map.accumulator.writer.metrics]
             ... see below ...
    
         # define the indexer mechanism
         [graphite-proxy-map.accumulator.writer.indexer]
             ... see below ...
                          
         # HTTP API for metrics and associated options
         [graphite-proxy-map.accumulator.api.metrics]
              ... see api config doc ...

         # HTTP API for indexer and associated options
         [graphite-proxy-map.accumulator.api.indexer]
              ... see api config doc ...

         # TCP API for metrics and associated options
         [graphite-proxy-map.accumulator.tcpapi.metrics]
             ... see api config doc ...
    
         # TCP API for indexer and associated options
         [graphite-proxy-map.accumulator.tcpapi.indexer]
             ... see api config doc ...
        
         # drop marvel
         [[graphite-proxy-map.map]]
            substring=".marvel"  # ignore elastic search marvel stuff
            reject=true


## SubWriter: Secondary writers

Currently there is the ability to add 2 different writers for a given accumulator.  Any more then 2 I've found to basically
be "too much" to handle (at high loads).  The config is the same as for the normal writers .. an abbreviated config skeleton is below.

Given the nature of the writer, you may need to specify a different "cache" for the writer.  For instance mixing 
`driver="cassandra-log-triggered"` and `driver="whisper"` cannot share the same cache mechanism as whisper does not
deal with the binary series blobs.  In the API section you can choose which cache to use.

This mode is good for a few scenarios.  

- Writing real data to cassandra as well as farming raw metrics to kafka.
- Dual writing to cassandra, and whisper (disk) when you need to back fill an existing graphite install.
- Writing data to some data store, and farming CSV file chunks to S3, etc.




    [graphite-proxy-map]
    listen_server="graphite-proxy" # which listener to sit in front of  (must be in the main config)
    default_backend="graphite-proxy"  # failing a match go here
    
        [graphite-proxy-map.accumulator]
         .... base options ...
         
         # A write back caching object is required. (yes there can be multiple, explained below)
         [[graphite-proxy-map.accumulator.writer.caches]]
             name="mygorillacache"
             ... see below ... 
             
         # A write back caching object is required. (yes there can be multiple, explained below)
        [[graphite-proxy-map.accumulator.writer.caches]]
            name="protobufwhisper"
            ... see below ... 

        # Define the writer and associated options
        [graphite-proxy-map.accumulator.writer.metrics]
            cache="mygorillacache"
            driver="cassandra"
             ... see below ...
    
        # define the indexer mechanism
        [graphite-proxy-map.accumulator.writer.indexer]
             ... see below ...
                          
        # HTTP API for metrics and associated options
        [graphite-proxy-map.accumulator.api.metrics]
            cache="mygorillacache"
            ... see api config doc ...

        # HTTP API for indexer and associated options
        [graphite-proxy-map.accumulator.api.indexer]
              ... see api config doc ...

        # TCP API for metrics and associated options
        [graphite-proxy-map.accumulator.tcpapi.metrics]
             ... see api config doc ...
        
        # Define the SUB writer and associated options
        [graphite-proxy-map.accumulator.subwriter.metrics]
            cache="protobufwhisper"
            driver="whisper"
             ... see below ...
    
        [graphite-proxy-map.accumulator.subwriter.indexer]
             ... see below ...   
              
        # TCP API for indexer and associated options
        [graphite-proxy-map.accumulator.tcpapi.indexer]
             ... see api config doc ...
         
        [[graphite-proxy-map.map]]
            ....

            
### Writer: Cache Configs

Cache sections are one of the more important sections.  They are used for both throttling writes, data compression, 
and hot reading.  For the `series_encoding` see [timeseries](./timesearies.md).

An important thing to remember is that if you expect alot of unique metrics then you'll need at least
`bytes_per_metric` * `max_metrics` of RAM, or you'll OOM kill things.

Also remember that for non-triggered, non-whisper, and non-leveldb writers, each time gets its own cache so the RAM
requirements increase by the number of times.

NOTE: Currently the `kafka-flat`, `mysql-flat`, and `elasticsearch-flat` writers do NOT use write back caches.  However
you need to include the section in case in the future they do.  They currently do not use them as they support "batch"
writing, these maintain an internal buffer for the batches. And no cassandra does not support batching in the way we 
use it so `cassandra-flat` and `whisper-flat` use the write back cache.


#### example for a cache for a whisper and cassandra-flat writers

    [[graphite-proxy-map.accumulator.writer.caches]]
    
    # a Name is needed so we can re-use the same cache for multiple things
    name="iamgorilla" 
    
    # Internal encoding: protobuf, gob, msgpack, json, gorilla 
    series_encoding="protobuf"
    
    # Max Unique Metrics: 
    # number of unique metrics allowed in the cache before we are forced to start dopping things
    # this makes sure we don't OOM kill ourselves.
    max_metrics=1024000  
    
    # Max Bytes per metric:
    # per unique store at most this number of bytes
    bytes_per_metric=1024
    
    # OverFlow method:
    # if we hit bytes_per_metric for a given metric, there are 2 options, drop any new incoming or force it to be writeen
    # overflow_method = `drop` drop any new metrics if the buffer is too big for that metric
    # overflow_method = `chan` FORCE this metric to be written to the writer
    # Not all writer backends support the `chan` method.   
    # NOTE: Currently only `whisper` and `cassandra-flat` support this method
    overflow_method="drop" 
    
    # Low Fruit Rate: This option is for the Whisper writer only currently
    # 
    # for non-series metrics, the typical behavior is to flush the highest counts first,
    # but that may mean lower counts never get written, this value "flips" the sorter at this % rate to
    # force the "smaller" ones to get written more often
    # the value says that 10% of each sorting calculation we flip the sorted list and pick the lowest counts
    
    low_fruit_rate= 0.10
    
       
#### example for a cassandra log triggered writer

    [[graphite-proxy-map.accumulator.writer.caches]]
    
    # a Name is needed so we can re-use the same cache for multiple things
    name="iamgorilla" 
    
    # Internal encoding: protobuf, gob, msgpack, json, gorilla 
    series_encoding="gorilla"
    
    # Max Unique Metrics: 
    # number of unique metrics allowed in the cache before we are forced to start dopping things
    # this makes sure we don't OOM kill ourselves.
    max_metrics=1024000 
    
    # Max Bytes per metric: not used for log based caches, log method is time based not size based
    # bytes_per_metric=1024
    
    # Low Fruit Rate: not used for here
    #low_fruit_rate= 0.10
       

#### example for a non-log cassandra writer

    [[graphite-proxy-map.accumulator.writer.caches]]
    
    # a Name is needed so we can re-use the same cache for multiple things
    name="iamgorilla" 
    
    # Internal encoding: protobuf, gob, msgpack, json, gorilla 
    series_encoding="gorilla"
    
    # Max Unique Metrics: 
    # number of unique metrics allowed in the cache before we are forced to start dopping things
    # this makes sure we don't OOM kill ourselves.
    max_metrics=1024000 
    
    # Max Bytes per metric: when a metric hits this size, it will be written to the DB
    bytes_per_metric=1024
    
       
## Writer: Metric Writer Configs

Each writer can have a slew of options available to them.  Below we try to enumerate them all. There are defaults provided
for each option that are reasonable.  The defaults are what are defined below.

### LevelDB

There is currently only the "leveldb-log" series writer available.  For LevelDB only the minimum resolution is stored
There are no rollups or multiple resolutions supported. 


    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
    
        # the name an be referenced in the api and tcp api sections as `use_metrics` 
        name="leveldb-metrics"
        
        driver="leveldb"
        dsn="/path/to/storage"
        
        [graphite-proxy-map.accumulator.writer.metrics.options]
            
            # size of the internal read cache block size (in MB)
            read_cache_size=8
            
            # size a levelDB sstable file can be before (in MB)
            file_compact_size=20
            
            # If 'true' this will create the Indexer tables as well as the metric tables
            # you can set this to false if your indexer is also NOT a levelDB indexer set
            open_index_tables = true
            
            ## Log options for caches, only applies to `-log` based writers
                    
            # Time in minutes that a chunk is written to before it is flushed to the metric tables
            cache_time_window=10
            
            # the number of chunks of time `cache_time_window` to store in RAM
            # there are `cache_num_chunks`+1 chunks when full the `+1` is for the current chunk
            cache_num_chunks=6
            
            # for the log, we flush out the state every `cache_log_flush_time` seconds
            cache_log_flush_time = 10
            
            # if this is true, then no chunk writting is performed, but the LOGs are still written
            # allows for "hot" spares (however, currently there is no mechanism to change this setting live.
            # that will change at somepoint).  So if you have 2 writers, one is the cache_only=false and cache_only=true
            # you'd need to change this setting are restart cadent in case you wanted to flip whos the primary writer.
            cache_only = false

### Redis

Fast (i really do love the redis), but if the metric volume is huge (~10k-40k writes per second GC can hurt, there's alot of strings/bytes to make).
I Recommend for this for when you need to read fast, and have a relatively low volume (in the 10k range) of metrics/second.  Make sure your TTLs are
 also decently small (hours, day) otherwise you'll soon nuke all the RAM you have.  Currently only the "flat" representation is available.  And no "indexing"
 So either LevelDB or ElasticSearch is still your best bet here.


Config for `redis-flat`.

    [graphite-proxy-map.accumulator.writer.metrics]
        name = "redis-metrics"
        driver = "redis-flat"
        dsn = "127.0.0.1:6379"
        cache = "protobuf"

        # since this writer can get really backedup on huge loads (ESes indexing fault)
        # we really need to limit things and start to drop otherwise, things will spiral to death
        input_queue_length = 10024

        [graphite-proxy-map.accumulator.writer.metrics.options]
        batch_count=1000

        # keys are prefixed with this value
        # keep an eye on the two "index like" keys `metric:uids` and `
        metric_index = "metrics"

        # concurrent writers (each has it's own connection pool for the pipline writers)
        workers = 8

        # database to use
        db = 0

        # password
        # password=""

        # regardless of a full batch push the batch at this interval
        periodic_flush="1s"

        # hourly, daily, weekly, monthly, none (default: hourly)
        # i would not use the "none" option unless you know what you're doing
        # keys will be prefixed by {metric}:{resolution}:{thisdate}:{uid}
        # as ZSETs using "time" (unix epoch) as the score for nicely orer items
        index_by_date="hourly"

 
            
### Cassandra

Config for `cassandra-flat` and `cassandra` (i.e. cassandra series) writers 

    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        
        # the name an be referenced in the api and tcp api sections as `use_metrics` 
        name="cass-flat-metrics"
    
        driver="cassandra-flat"
        dsn="hostname,otherhostname"
        port=9042
        
        # which cache to use from the [.cache] list
        cache="my-series-cache"
        
        [graphite-proxy-map.accumulator.writer.metrics.options]
            
            # use this keyspace
            keyspace="metric"
            
            # metrics table name/prefix
            metrics_table="metrics"
            
            # path table for index
            path_table="path"
            
            # segment table for index
            segment_table="segment"
 
            # id table for index
            id_table="ids"
            
            # cassandra write consistency: one, local_quorum, quorum
            write_consistency="one"
            
            # cassandra read consistency: one, local_quorum, quorum
            read_consistency="one"            
            
            # tells the start to find all cass node before accepting sessions
            # this can take a long time espcially if there are lots of nodes, ip alias, peering alias, or hostname alias
            # the default is true, but false is also ok (sometimes nessesary if you want things to start up fast)
            sniff=true
            
            # duration of any query timeouts
            timeout="30s"   
            
            # authentication
            user=""
            password=""
            
            # compress traffic in and out of things
            compress=true
            
            # Number of connections in the cassandra pool
            numcons=10
            
            # number of workers that do writes
            write_workers=32
            
            # number of items in the dispatch queue before things block
            write_queue_length=102400
            
            # how fast can we write .. this is a rate limiter to keep things from 
            # going totally nuts and either hurt cassandra and cadent
            writes_per_second=5000
            
            # Use tables per resolution defined in the times=[...] section
            # if true, and times=["1s", "10s", "1m"] then three tables are made/should exist
            # {metrics_table}_1s, {metrics_table}_10s, and {metrics_table}_60s
            # this mode is highly recommended for large clusters with TTLs, as the compaction
            # works better and faster on smaller tables, and the compaction can be tuned per table to handle 
            # the "tombstone" issue on select queries.
            # the tables prefixes are in seconds.
            table_per_resolution=false
            
            # disable/enable "sniffing" this is a start up process that attempts to get the topology for the cassandra
            # nodes.  If disabled, it can lead to faster startup, at the expense of an initial few min of higher load
            # while things may not be using the Token aware policy effectively (eventually things will be figured out)
            sniff=true
            
            # all cassandra selection policies are "token aware" meaning it will attempt to insert/get elements
            # that are on the real cassandra node in question (rather then letting cassandra act as a proxy to
            # pick the proper one).  In case the token aware cannot function as planned this is the backup policy.
            #
            # epsilon_greedy = pick the host that is the most responsive (this can be a CPU intensive operation at times)
            # host_pool = pick the host that least used from a pool
            # round_robin = cycle through hosts
            #
            backup_host_policy="host_pool"
            
 
Config options for `cassandra-log` writer
    
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        ... same as `cassandra` ...
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
        ... same as `cassandra` with the follwing additions ...   
        
        # the Log table we use
        log_table="metriclogs"
        
        # writer index (required)
        # the Log table is named {log_table}_{writer_index}_{min resolution}s
        # since there can be mulitple writers, their logs are not the same, and we segment out each writer's
        # log based on this index
        writer_index=1
        
        ## Log options for caches, only applies to `-log` based writers
        
        # Time in minutes that a chunk is written to before it is flushed to the metric tables
        cache_time_window=10
        
        # the number of chunks of time `cache_time_window` to store in RAM
        # there are `cache_num_chunks`+1 chunks when full the `+1` is for the current chunk
        cache_num_chunks=6
        
        # for the log, we flush out the state every `cache_log_flush_time` seconds
        cache_log_flush_time = 10
        
        # if this is true, then no chunk writting is performed, but the LOGs are still written
        # allows for "hot" spares (however, currently there is no mechanism to change this setting live.
        # that will change at somepoint).  So if you have 2 writers, one is the cache_only=false and cache_only=true
        # you'd need to change this setting are restart cadent in case you wanted to flip whos the primary writer.
        cache_only = false
                     
         
Config options for the `cassandra-triggered` writer
 
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        ... same as `cassandra` ...
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
        ... same as `cassandra` with the following additions ...           
        
        # Byte size for any rollups created before we add a new series
        # This will defaul to the CACHE byte size if not specified
        # rollup_byte_size=1024 
        
        # the Rollup method: `blast` or `queue`
        # the BLAST method is choosen for this writer as writes for the overflow style tend to be 
        # relatively low and constant, thus the rollups as well are low and constant
        rollup_method="blast"
        
        ### BLAST method options
        # BELOW are the options if in "blast" mode
        # number of workers to eat the dispatch queue
        rollup_workers=8
        
        # dispatch queue length before block
        rollup_queue_length=16384
        
        #### QUEUE method options
        # a kinder gentler rollup process
        rollup_queue_workers=4
                
        # dispatch queue length before block
        # rollup_queue_length=128
        
        
Config options for the `cassandra-log-triggered` writer
  
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        ... same as `cassandra` ...
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
         ... same as `cassandra` with the following additions ...           
         
         # Byte size for any rollups created before we add a new series
         # since the cache here is "no sized based" we can override the size in the cache as
         # that setting is useless in the `log` case
         rollup_byte_size=2048 
         
         # the Rollup method: `blast` or `queue`
         # the QUEUE method is choosen as the default for this writer as write here happen
         # in large blasts when the chunk time has expired, which if in blast mode would cause
         # a huge spike in CPU/IO/etc for everything as we attempt to compute the thousands/millions
         # of rollups.  If the blast mode does not complete before the next time chunk we wind up in a death spiral.
         # the queue method does not add a new rollup if the metric is already queued, so we never hit this
         # issue
         rollup_method="queue"
         
         # a kinder gentler rollup process
         rollup_queue_workers=4
                 
         # queue length before block
         rollup_queue_length=128
         

### MySQL

For mysql all metrics are stored in a `table_per_resolution` sense (see cassandra config above). So There will be tables of the form

    {table}_{res}s
    
    i.e for times=["10s", "1m", "10m"]
    
    metrics_10s
    metrics_60s
    metrics_600s
        
Config options for the `mysql-flat` writer
  
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
    # the name an be referenced in the api and tcp api sections
    name="my-flat-metrics"
    
    driver = "mysql-flat"
    dsn = "$ENV{MYSQL_USER:admin}:$ENV{MYSQL_PASS:}@tcp($ENV{MYSQL_HOST:localhost}:3306)/$ENV{MYSQL_DATABASE:cadent}"
    
    # which cache to use from the [.cache] list
    cache="my-series-cache"
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
         
         # the main metrics table
         table="metrics
         
         # table for path index
         path_table="metric_path"
         
         # table for segments index
         segment_table="metric_segment"
         
         # table for tags
         tag_table="metric_tag"
         
         # table for cross ref tags
         tag_table_xref="metric_tag_xref"
         
         # Size of the number of items in a batch insert
         batch_count=1000
         
         # every "periodic_flush" force a batch write 
         periodic_flush="1s"
         
        
Config options for the `mysql` writer
  
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        ... same as `mysql-flat` above ...
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
         
         # the main metrics table
         table="metrics
         
         # table for path index
         path_table="metric_path"
         
         # table for segments index
         segment_table="metric_segment"
         
         # table for tags
         tag_table="metric_tag"
         
         # table for cross ref tags
         tag_table_xref="metric_tag_xref"
         
         # number of workers that do writes
         write_workers=16
         
         # number of items in the dispatch queue before things block
         write_queue_length=102400
         
         # Since mysql does not support TTLs directly, run a little job every 5m that will
         # delete any "expired" metric series from the DB, this only matters if the TTL is specified
         # in the `times` array
         expire_on_ttl=true
         
Config options for the `mysql-triggered` writer
  
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        ... same as `mysql` above ...
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
        ... same as `mysql-flat` above ...
        
        # Byte size for any rollups created before we add a new series
        # This will defaul to the CACHE byte size if not specified
        # rollup_byte_size=1024 
        
        # the Rollup method: `blast` or `queue`
        # the BLAST method is choosen for this writer as writes for the overflow style tend to be 
        # relatively low and constant, thus the rollups as well are low and constant
        rollup_method="blast"
        
        ### BLAST method options
        # BELOW are the options if in "blast" mode
        # number of workers to eat the dispatch queue
        rollup_workers=8
        
        # dispatch queue length before block
        rollup_queue_length=16384
        
        #### QUEUE method options
        # a kinder gentler rollup process
        rollup_queue_workers=4
                
        # dispatch queue length before block
        # rollup_queue_length=128
        
### Kafka

The flat writer simply pushes a message of the form

    message SingleMetric {
    	uint64 id;
    	string uid;
    	int64 time;
    	string metric;
    	double min;
    	double max;
    	double last;
    	double sum;
    	int64 count;
    	uint32 resolution;
    	uint32 ttl;
    	[]Tag tags;
        []Tag  meta_tags;
    }

THe series writer pushes a message of the form
    
    message SeriesMetric {
    	uint64 id;
    	string uid;
    	int64 time;
    	string metric;
    	string encoding;
    	bytes data;
    	uint32 resolution;
    	uint32 ttl;
    	[]Tag tags;
    	[]Tag meta_tags;
    }
        
Config options for the `kafka` and `kafka-flat` writer
  
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
    driver = "kafka" # or `kafka-flat`
    dsn = "localhost:9092,otherhost:9092"
    
    # which cache to use from the [.cache] list
    cache="my-series-cache"         
         
    [graphite-proxy-map.accumulator.writer.options]
        
        # Which topic to publish to
        metric_topic="cadent"
        
        # Messages can be one of 3 forms, json, msgpack, or protobuf
        encoding="json"
        
        # How many message to push in a kafka batch 
        batch_count=1024
        
        # How many times to retry if failure
        max_retry=10
        
        # Kafka acknolegement type
        # `local` : just one node
        # `all` : wait for all replicas to ack as well
        ack_type="local"
        
        # transport compression
        # can be "none", "snappy", "gzip"
        compression="snappy"
        
        # how often to flush the message batch queue
        flush_time="1s"
        
### ElasticSearch

Currently only the "flat" writer is supported.  Like MySQL the metric "indexes" are prefixed with the resolution

    {metric_index}_{res}s
        
    i.e for times=["10s", "1m", "10m"]
    
    metrics_10s
    metrics_60s
    metrics_600s

Flat writer config options

    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
    driver = "elasticsearch-flat" 
    dsn = "http://localhost:9200,http://otherhost:9200"
    
    # which cache to use from the [.cache] list
    cache="my-series-cache"         
         
    [graphite-proxy-map.accumulator.writer.options]
        
        # authentication
        user=""
        password=""
        
        # tells the start to find all ES node before accepting sessions
        # If your ES cluster is behind a load balencer you should set this to false
        sniff=false
        
        # the index in ES for metrics
        metric_index="metrics"
        
        # DEBUG MODE (don't use this in production) things will log every `curl` it performs
        enable_traceing=false
        
        # Size of the number of items in a batch insert
        batch_count=1000
        
        # every "periodic_flush" force a batch write 
        periodic_flush="1s"
        
        # create indexes by metric date
        # hourly, daily, weekly, monthly (and none) are supported
        # please note, you'll need to allow the `metric_index`* indexes to be autocreated
        # and if you change this after running, old indexes will no longer be used
        index_by_date="none"
        
        
        
### Whisper

This acts like good old carbon-cache, and writes whisper files.  The `writes_per_second` is very important to 
tune to your available "iOPS" operations.  A good rule is allocate 60% i.e. if your iOPS limit is == 3000, then choose 2000.


Flat writer config options

    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
    
    # the name an be referenced in the api and tcp api sections
    name="whis-metrics"
    
    driver = "whisper" 
    dsn = "/path/to/storage"
    
    # which cache to use from the [.cache] list
    cache="my-series-cache"         
         
    [graphite-proxy-map.accumulator.writer.options]
        
        # This setting is internal to the whisper file itself.
        # this is the DEFAULT setting for counter types, if it's detected that the metric
        # is a SUM (counters), xFilesFactor will be AUTO set to 0.0
        # is a MIN/MAX (upper/lower), xFilesFactor will be AUTO set to 0.1, see the whisper documentation for more detais
        xFilesFactor=0.3
        
        # number of workers that do writes
        write_workers=16
        
        # number of items in the dispatch queue before things block
        write_queue_length=102400
        
        # how fast can we write .. this is a rate limiter to keep things from 
        # going totally nuts and either hurt cassandra and cadent
        writes_per_second=2000
        
        
### RAM
            
This "writer" does not actually write anything, just holds chunks in RAM

Config options for `ram-log` writer
    
    # Define the writer and associated options
    [graphite-proxy-map.accumulator.writer.metrics]
        
        # the name an be referenced in the api and tcp api sections
        name="ram-metrics"
        
        driver="ram-log"
    
    [graphite-proxy-map.accumulator.writer.metrics.options]
        
        # Time in minutes that a chunk is written to before it is flushed from ram
        cache_time_window=10
        
        # the number of chunks of time `cache_time_window` to store in RAM
        # there are `cache_num_chunks`+1 chunks when full the `+1` is for the current chunk
        cache_num_chunks=6
        
      
       
## Writer: Index Writer Configs

Indexers are separate from the metric writers, however, you need a metrics writer that will farm index writes to 
the indexer modules.  Configuration is a little less involved here, and most indexers share the same set of options.


### LevelDB

    [graphite-proxy-map.accumulator.writer.indexer]
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="level-indexer"
        
        driver="leveldb"
        dsn="/path/to/storage"
    
    [graphite-proxy-map.accumulator.writer.indexer.options]
    
        # hold a list of things we've already indexed so we don't keep repeating ourselves
        # the size of this list, if we hit this limit, we stop adding elements to the list, and drop it
        cache_index_size=1048576
        
        # number of things to write a second, a rate limiter to not overwhelm either disk or DB backend
        writes_per_second=1000
        
        # how many workers to consume the dispatch queue
        write_workers=8
        
        # size of the dispatch queue before we block
        write_queue_length=1048576
        

### Cassandra

    [graphite-proxy-map.accumulator.writer.indexer]
    
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="cass-indexer"
                
        driver="cassandra"
        dsn="hostname,otherhostname"
        port=9042
    
    [graphite-proxy-map.accumulator.writer.indexer.options]
        
        # assign things written to by this node as the writer
        # if you have multiple writers you'll need to have UNIQUE ids for them (or certainly should)
        writer_index=1
                
        # a machine local index is also kept on the host in question
        # to act as a ram/leveldb based indexer for speed purposes 
        # This index is "wiped out" on each restart of cadent and re-populated from the incoming
        # or pre-poulated from a backfilling operation from the main storage DB itself
        local_index_dir="/tmp/cadent_index"
        
        # use this keyspace
        keyspace="metric"
                
        # path table for index
        path_table="path"
        
        # segment table for index
        segment_table="segment"
        
        # cassandra write consistency: one, local_quorum, quorum
        write_consistency="one"
        
        # cassandra read consistency: one, local_quorum, quorum
        read_consistency="one"            
        
        # tells the start to find all cass node before accepting sessions
        # this can take a long time espcially if there are lots of nodes, ip alias, peering alias, or hostname alias
        # the default is true, but false is also ok (sometimes nessesary if you want things to start up fast)
        sniff=true
        
        # duration of any query timeouts
        timeout="30s"   
        
        # authentication
        user=""
        password=""
        
        # compress traffic in and out of things
        compress=true
        
        # Number of connections in the cassandra pool
        numcons=10
        
        # hold a list of things we've already indexed so we don't keep repeating ourselves
        # the size of this list, if we hit this limit, we stop adding elements to the list, and drop it
        cache_index_size=1048576
        
        # number of things to write a second, a rate limiter to not overwhelm either disk or DB backend
        writes_per_second=1000
        
        # how many workers to consume the dispatch queue
        write_workers=128
        
        # size of the dispatch queue before we block
        write_queue_length=1048576
        
### MySQL

    [graphite-proxy-map.accumulator.writer.indexer]
    
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="my-indexer"
        
        driver = "mysql"
        dsn = "$ENV{MYSQL_USER:admin}:$ENV{MYSQL_PASS:}@tcp($ENV{MYSQL_HOST:localhost}:3306)/$ENV{MYSQL_DATABASE:cadent}"
         
    [graphite-proxy-map.accumulator.writer.indexer.options]
        
        # a machine local index is also kept on the host in question
        # to act as a ram/leveldb based indexer for speed purposes 
        # This index is "wiped out" on each restart of cadent and re-populated from the incoming
        # or pre-poulated from a backfilling operation from the main storage DB itself
        local_index_dir="/tmp/cadent_index"       
        
        # path table for index
        path_table="path"
        
        # segment table for index
        segment_table="segment"
        
        # table for tags
        tag_table="metric_tag"
         
        # table for cross ref tags
        tag_table_xref="metric_tag_xref"
        
        # hold a list of things we've already indexed so we don't keep repeating ourselves
        # the size of this list, if we hit this limit, we stop adding elements to the list, and drop it
        cache_index_size=1048576
        
        # number of things to write a second, a rate limiter to not overwhelm either disk or DB backend
        writes_per_second=1000
        
        # how many workers to consume the dispatch queue
        write_workers=128
        
        # size of the dispatch queue before we block
        write_queue_length=1048576
        
### ElasticSearch

    [graphite-proxy-map.accumulator.writer.indexer]
    
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="es-indexer"

        driver = "elasticsearch"
        dsn = "http://localhost:9200,http://otherhost:9200"
         
    [graphite-proxy-map.accumulator.writer.indexer.options]
        
        # assign things written to by this node as the writer
        # if you have multiple writers you'll need to have UNIQUE ids for them (or certainly should)
        writer_index=1
                    
        # a machine local index is also kept on the host in question
        # to act as a ram/leveldb based indexer for speed purposes 
        # This index is "wiped out" on each restart of cadent and re-populated from the incoming
        # or pre-poulated from a backfilling operation from the main storage DB itself
        local_index_dir="/tmp/cadent_index"       
                
        # authentication
        user=""
        password=""
        
        # path index
        path_index="metric_path"
        
        # segment index
        segment_table="metric_segment"
        
        # tag index
        tag_index="metric_tag"
        
        # hold a list of things we've already indexed so we don't keep repeating ourselves
        # the size of this list, if we hit this limit, we stop adding elements to the list, and drop it
        cache_index_size=1048576
        
        # number of things to write a second, a rate limiter to not overwhelm either disk or DB backend
        writes_per_second=200
        
        # how many workers to consume the dispatch queue
        write_workers=8
        
        # size of the dispatch queue before we block
        write_queue_length=1048576
        
        # The max number of results we will pull from any Find query
        max_results=1024
        
### Kafka

Push an "index" message to a topic.
    

    [graphite-proxy-map.accumulator.writer.indexer]
        driver = "kafka"
        dsn = "localhost:9092,otherhost:9092"
         
    [graphite-proxy-map.accumulator.writer.indexer.options]
       
        # If false this becomes a "noop" basically
        write_index=true
       
        # path index
        index_topic="cadent"
        
        # Messages can be one of 3 forms, json, msgpack, or protobuf
        encoding="json"
        
        # How many message to push in a kafka batch 
        batch_count=1024
        
        # How many times to retry if failure
        max_retry=10
        
        # Kafka acknolegement type
        # `local` : just one node
        # `all` : wait for all replicas to ack as well
        ack_type="local"
        
        # transport compression
        # can be "none", "snappy", "gzip"
        compression="snappy"
        
        # how often to flush the message batch queue
        flush_time="1s"


        
### Whisper

The "whisper" indexer actually does not do any writing, but is used for the API querying, as whisper files are
basically "glob" from on disk file system
    
     [graphite-proxy-map.accumulator.writer.indexer]
    
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="whis-indexer"
        
        driver = "whisper"
        dsn = "/path/to/whisper/files"


        
### Ram

The "ram" indexer (pseudo ram) forms the basis for a few other indexers.  
This is the basis for several other indexers (cassandra, whisper, elastic, mysql).  Is both a RAM index
as well as a modified leveldb index on disk.  Like Ram it's ephemeral (in that it's wiped out each restart)
    
     [graphite-proxy-map.accumulator.writer.indexer]
    
        # the name an be referenced in the api and tcp api sections as `use_indexer`
        name="ram-indexer"
        
        driver = "ram"
        [graphite-proxy-map.accumulator.writer.indexer.options]
        
            # a machine local index is also kept on the host in question
            # to act as a ram/leveldb based indexer for speed purposes 
            # This index is "wiped out" on each restart of cadent and re-populated from the incoming
            # or pre-poulated from a backfilling operation from the main storage DB itself
            local_index_dir="/tmp/cadent_index" 



### Noop

A "do nothing" indexer
    
     [graphite-proxy-map.accumulator.writer.indexer]
         driver = "noop"

       
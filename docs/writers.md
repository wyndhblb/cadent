
# Writers 

Accumulators can "write" to something other then a tcp/udp/http/socket, to say things like a FILE, MySQL DB or cassandra.
(since this is Golang all writer types need to have their own driver embed in).  If you only want accumulators to write out to
these things, you can specify the `backend` to `BLACKHOLE` which will NOT try to reinsert the line back into the pipeline
and the line "ends" with the Accumulator stage.

    InLine(port 8126) -> Splitter -> [Accumulator] -> WriterBackend

Writers should hold more then just the "aggregated data point" but a few useful things like

    TimeStamp, Min, Max, Sum, Last and Count

because who's to say what you really want from aggregated values.
`Count` is just actually how many data points arrived in the aggregation window (for those wanting `Mean` = (Sum / Count))

If using the `bypass` mode (see [bypass](#bypass) below), then a metric only contains

    TimeStamp, Count = 1, Sum = {value}

Writers themselves are split into 2 sections "Indexers" and "Metrics"

Indexers: take a metrics "name" which has the form

    Tag{
        Name string
        Value string
    }

    StatName{
        Key string
        UniqueId uint64
        UniqueIdString string
        Tags  []Tag
        MetaTags []Tag
        Resolution uint32
        TTL uint32
    }

And will "index it" somehow.  Currently the "tags" are not yet used in the indexers .. but the future ....

The StatName has a "UniqueID" which is basically a FNV64a hash of the following

    FNV64a(Key + ":" + sortByName(Tags))
   
If `tag_mode=all` then the key becomes

     # if there are MetaTags
     
     FNV64a(Key + ":" + sortByName(Tags) + ":" + sortByName(MetaTags))
     
     # if not
     
     FNV64a(Key + ":" + sortByName(Tags))


Tags therefore will be part of the unique identifier of things.  Several formats may include a "key + tags" or just "tags".
This can cause a little confusion as to what the unique ID is.  For instance in a graphite standard, the key is
 `pth.to.my.metric`.  In the [metrics2.0](https://github.com/metrics20/spec) spec there is no "key" so
 the acctuall key is derived inside cadent as `name1=value1.name2=value2...` but the unique ID would then be

    FNV64a("name1_is_value1.name2_is_value2..." + ":" + "name1=value1 name2=value2 ..")

So it's a bit of doubling up on the key + tags set, but for systems that do both a "key" and tags then
this method provides some resolution for differeing writer backend systems.

MetaTags are no considered part of the unique metric, as they are subject to change.  An example of a MetaTag is the
sender of the metrics (statsd, cadent, diamond, collectd, etc).  Where the metric itself is the same, but the sender
has changed.

It should be noted that any cross MetaTag relations will therefore not be strictly unique.  For example if you change
from diamond to collectd, the metric will effectively be tagged with both.

Metrics: The base "unit" of a metric is this

    repr.StatRepr{
        Time int64
        Min float64
        Max float64
        Last float64
        Sum float64
        Count int64
    }


Internally these are stored as various "TimeSeries" which is explained in [the timeseries doc](./timeseries.md)
, but basically some list of the basic unit of `StatRepr`.

When writing "metrics" the StatName is important for the resolution and TTL as well as its UniqueID.

A "total" metric has the form

    TotalMetric{
         Name repr.StatName
         Metric repr.StatRepr
    }

If using timeseries, the object is then

    metrics.TotalTimeSeries{
        Name Statname
        Series []repr.StatRepr  # (this is different for every series type)
    }
    


## Schemas

### You should check the injected schemas

Cadent injects the "basic" schemas for MySQL, ElasticSearch, and Cassandra.

But as a long time ops person, not every schema is geared towards use cases. So please check them to make sure they are what you
really want.

The schemas presented below are what Cadent expects in it's tables, so one will at least need to match them in some form
For MySQL, for instance, if you wanted to do TTLs on data, you would need to partition the table to allow for easy
dropping of data at the proper time (and thus some of your indexes may change).  Cassandra is a bit more tricky as the
query patterns expect some element of consistency in the Primary key, but you may want different replication,
drivers, and other options that make your environment happy.



## Status

Not everything is "done" .. as there are many things to write and verify, this is the status of the pieces.

| Driver   | IndexSupport  |  TagSupport |  SeriesSupport | LineSupport | MapSupport | TriggerSupport |
|---|---|---|---|---|---|---|
| cassandra | write+read  | No  | write+read | write+read | read+write | Yes |
| mysql  | write+read  | write+read  | write+read  | write+read  | No | Yes |
| kafka  |  write | write | write  | write  | n/a | n/a | 
| elasticsearch |  read+write | read+write | No | read+write | read+write | No |
| whisper|  read | n/a | n/a  | write+read |  n/a | n/a |
| leveldb |  write+read | No |  write+read | No | n/a |  n/a |
| file |  n/a | n/a | n/a  | write |  n/a | n/a |
| ram  | No | No | read+write(ish) | n/a | n/a | n/a |
| redis  | No | No | No | read+write (maps) | read+write | n/a | 



`IndexSupport` means that we can use this driver to index the metric space.

`TagSupport` means the driver will be able to index tags and read them (write/read).

`SeriesSupport` means the driver can support the TimeSeries binary blobs.

`LineSupport` means the driver can write and/or read the raw last/sum set from the Database backend.

`TriggerSupport` is the rollup of lower resolution times are done once a series is written.

`MapSupport` is an alternative to `LineSupport` in that the storage mechanism is pivoted.

`n/a` implies it will not be done, or the backend just does not support it.

`No` means it probably can be done, just not complete.


### Below are the "driver" names to use in configs for each backing DB (if they exist)

| DataBase   |  IndexName | LineName | MapName | SeriesName | TriggerName | LogSeriesName |
|---|---|---|---|---|---|---|
| cassandra | cassandra | cassandra-flat |cassandra-flat-map | cassandra | cassandra-triggered | cassandra-log{-triggered} |
| mysql  | mysql | mysql-flat | | mysql | mysql-triggered |  |
| kafka  | kafka | kafka-flat | | kafka |  |  |
| elasticsearch | elasticsearch | elasticsearch-flat | elasticsearch-flat-map |  |  |  |
| whisper |  whisper | whisper | | | | |
| RAM |   | | | | ram-log | |
| leveldb | leveldb | | | | leveldb-log | |
| file |  | file | | | | | 
| redis |  | redis-flat | redis-flat | | | |


## "Map" Storage

Some DBs (cassandra, elasticsearch, and redis) support updates to single columns/documents in a Map or List like fashion.
This is a basically very close to storing data like the Series, but the series are not "compressed" and data is written as 
 fast as it can.  
 
In cassandra we are storing something like

   `ID, SLAB, map<time, points>`
   
`SLAB` is an hours worth by default (YYYYMMDDHH), but can be daily, weekly, monthly, yearly if you data input warrants it.

For ElasticSearch we acctually store a list in a field

    `[ [time, min, max, last, sum, count], ... ]`  

as we also store a more compact form `[time, sum]` for things that only have one entry in a given resolution period.

For Redis we use ZSETs of the form where the key is stringified metric valu(s) and the score is the time, so we can alwasy get back
Sorted lists.  This does require a bit of decoding step.  

    `metric:SLAB:id => [time:min:max:last:sum:count' : int(time), ...]`
    
`time` is included in the key as it saves a few CPU cycles (i.e. faster) on the decode step and the read from redis step.

*Don't ever use anything other then an hour if you are interested data that happens every "x" seconds* 
 
This method can lead to increased Read and Writing efficiency depending on the typical query window and data volume.

This does create some "tombstone" issues for both ElasticSearch and Cassandra, in that for an hour of 1 second data, you can have
up to 3600 ish "dead" documents that need garbage collecting from the DB itself.

I highly recommend this method for ElasticSearch if needed, as the update document API is much more efficient then the insert.

For the redis storage (the item that inspired this technique), this is actually the method for the "Flat" engine as well, 
as to query any sort of volume requires this structure.

## When to choose and why

The main writers are

    - cassandra: a binary blob of timeseries points between a time range

        Good for the highest throughput and data effiency for storage.

    - cassandra-triggered: a binary blob of timeseries points between a time range

       Same as `cassandra` but uses triggering for rollups (see below)

    - cassandra-log: a binary blob of timeseries points between a time range put with a periodic "log"

           Same as `cassandra` for series, but uses a log (see #Log below)

    - cassandra-log-triggered: same as `cassandra-triggered`, but using the log technique as well

    - ram-log: same chunked cache mechanism as the cassandra-log, but just don't really write anything

           All metrics queries are just for what's in the cache

    - cassandra-flat: a row for each time/point

        Good for simplicity, and when you are starting out w/ cassandra to verify things are working as planned
        But, not exactly space efficient nor read/api performant.  But may be usefull (as it was for me)
        in verifification of the internals. As well as if you are going to have "out-of-order" time series.

    - cassandra-flat-map: a row for each time slab

        A more efficient storage plan.  Rather then sore each point, append a list of points to 
        a column over a "time slab" (YYYYMMDDHH, YYYYMMDD, etc).  Basically an entire hour is store in one column that
        is appended to.  For queries that are in the time range of the slab delta.  Writes tend to be a little faster
        as well as reads as we don't have to scan alot of rows.  This is a good schema for higher volume metrics
        in a given time slab (an hour).  If your metrics are really sparse over that time trame, the cassandra-flat
        is probably better as it offers less "read" logic for other consumers.

    - whisper: standard graphite format

        Good for local testing or even a local collector for a given node, or if you simply want a better/faster/stronger
        carbon-cache like entity.

    - mysql: a binary blob of timeseries points between a time range

        If your data storage requirements are not overly huge, this is pretty good. (also since the local dev
        on cassandra is "hard" and slow as it was never meant to run in a docker container really, this is
        a bit easier to get going and start playing w/ blob formats)

    - mysql-triggered: a binary blob of timeseries points between a time range

        Same as `mysql` but uses triggering for rollups (see below)

    - mysql-flat:

        Like cassandra-flat but for mysql ..
        Good for "slow" stats (not huge throughput or volume as you will kill the DB)

    - kafka + kafka-flat:

        Toss stats into a kafka topic (no readers/render api for this mode) for processing by some other entity

    - elasticsearch-flat:

       For indexing, this is a good option especially if your cardinality of keys/tags is very high (where things
       benifit from a standard "text" indexer and not a scan of lists).

       ElasticSearch is not really efficient for huge volumes of time series data (unless you have a monster
       cluster).  MySQL acctually performs better for that, however it is, of course, not horizonally scaleable like ES.

    - elasticsearch-flat-map:

       Like the cassandra-flat-map (see above) and is much more scalable for higher metric volumes (however not as good
       as redis or cassandra for the "flat" based writers.

    - leveldb-log

        the Leveldb uses the same "log" mechanism (see below) and is a series writer. It does not support triggered rollups.
        Only ONE resolution is supported.

        This is good for local disk based installs (i.e. having a cadent on each node, and some "crawler" or external
        query engine hit each node for node specific items.

    - redis-flat
    
        Redis is one of the best fastest DBs around for smallish data sets (small meaning as much RAM as you have)
        This one is a fast insert/reader but cannot take too much data.  As it's flat writer (using ZSETs) data can get
        very large.  TTLs are important, as well as the number of time/value pairs + unique keys.  But for ~10k unique
        metrics on a decent TTL, this works really well.
        
        This uses the "map" technique using internally (ZSETs where "int64(time)" is the "score" for a given data point)

    - ram-log

        An in-ram-only writer, basically it's just the chunked cache, nothing else.  A LevelDB index is good choice
        for the indexer (a ram only indexer is in the works).


Some basic performance things:

Cassandra is by far the best option for large metric volumes, replication, and general read/write speed.

MySQL is good for local development or relatively small volumes of metrics. Replication is up to your MySQL install.
Reads are pretty fast, writes can be much slower simply due to the ACID nature of things.  It also supports tags
for indexing as well (due to MySQLs need groupby/having nature for complex queries it's not the fastest thing on
the planet, but if your cardinatlity is low, it should be ok).

ElasticSearch is good for indexing, and though it does have built in "aggregation" query forms, it does not
handle huge inputs effectively unless your cluster if very large.  Use this option if you have High cardinality
but low metric volume (things like when a user logs in, or other similar things).

If storage space is a concern, you should use the "series" writers, but you risk losing data on the event of a crash (i.e.
data that is still in ram and not yet written).  While the `log` based writers to aid w/ this, there is the time to
both repair a node and timw for the re-add-back the log that can result in loss from that node.


## ByPass

Bypassing means that the Accumulator section will NOT accumulate.  Things are sent DIRECTLY to writers as is.  
This can work for graphite like inputs where things are assumed to have been already accumulated (or not) via another
Cadent accumulator or external things.  So all data points go in as "count of 1" and "sum of {incoming val}".  

If using the Gorilla Time Series, this can work pretty well as data points are highly compressible.

DO NOT use bypass if you want a statsd like interface, no accumulation will be performed.  A good set up is

    Statsd Input -> Statsd Accumulate -> GraphiteBackend -> ByPass -> Writer

PLEASE NOTE:: 1 SECOND is the minimum resolution (currently), "highres" (or nanosecond) series are not yet ready
for production. So if you inject metrics fast, the raw series will retain each point, (with a time delta of 0), and
the API section will "merge" them into a 1second bin (so you still get the counts, last, min, max, etc).

PLEASE NOTE PART 2: Data is stored w/ the LOWEST resolution in the config as well, so if you use `bypass=true` and
your `times=[10s, 1m, 10m]` then (depending on which backend) things will be tagged w/ the 10s resolution or in the 
10s resolution table.

The API will use the min `times=[Xs...]` as the outgoing resolution, set things to 1s if you want the min full res.

Example config

    [graphite-proxy-map]
    listen_server="graphite-proxy" # which listener to sit in front of  (must be in the main config)
    default_backend="graphite-proxy"  # failing a match go here
    
        [graphite-proxy-map.accumulator]
        backend = "BLACKHOLE"
        input_format = "graphite"
        output_format = "graphite"
        random_ticker_start = false
    
        # if bypass is TRUE, then NO accumulation will happen, and things
        # will get directly sent "asis" to the Writers
        bypass=true
    
        # aggregate bin counts
        accumulate_flush = "1s"
        times = ["1s:24h", "10s:48h"]
    
        # cache objects to be shared (or not) across the writer backends
        [[graphite-proxy-map.accumulator.writer.caches]]
            name="gorilla"
            series_encoding="gorilla"
            bytes_per_metric=256
            max_metrics=1024000  
        
        ... the rest of the  writers, indexers, api, etc configs ...
        
        

## Indexing


This is probably the most tricky part to get "correct".  There are many possible ways to do this, but we need to rememeber
we have a few fundemental things we want to be able to do.

For more info on how things are passed around [see the uniqueid doc](./uniqueid.md).


### Metric Path

The first is to behave and act like good-old-graphite, where there are not "tags" and just "metric paths" of the form

    my.metric.is.fun

To query these we can simply split the paths by the `.` and index things accordingly.  Since the original graphite
pattern is basically to do file system globs, finding something like

    `my.m*.{is,are}.*`

can easily map to a file system glob, or a nice text based index file and a brute force regex can be done.

The trouble is that this is a very "local" experience, the index live on just the machine in question.  We would like some
redendency in the mix, so we need to port this behavior to various other Database systems.  We end up doing something like
an inverted index, which is an expensive operation to compute.

We take the metric path, break it up into each set and insert

    my.metric.is.fun ->
        segment=my hasdata=false base=my.metric.is.fun baselen=4
        segment=my.metric hasdata=false base=my.metric.is.fun baselen=4
        segment=my.metric.is hasdata=false base=my.metric.is.fun baselen=4
        segment=my.metric.is.fun hasdata=true base=my.metric.is.fun baselen=4

Now we can easily query the database for `my.m*.{is,are}.*` by doing

    foreach row in (select * from table where baselen=4 and segment="my"):
         if regex.Match(row.base):
             return row.base


## Tags

Tags add yet another layer to things.  Where we now can crawl the metric path tree, tags are effectively filtering
metric paths, and since we don't know the combo of tags before hand, this can lead to some large space scans

    my.metric.path{unit=jiff,instance=r3.xl} -> pretty narrow unless you have alot of r3.xls each with hundered of this metric
    my.metric.path{host=abc,unit=jiff} --> pretty narrow as we have the host, and a unit
    my.metric.path{host=abc} --> also pretty narrow (unless this is kafka and there will litteraly be 10000 of these per host)
    my.metric.path{type=cpu} --> can be HUGE, all cpu metrics for the materics key

Thus the issue becomes one of being able to effecitvely query all the metrics that may be in a tags subset (we can easily
be in the many thousands, which for alerting/graphing/just looking so down right silly.  But we still need some cross reference
tables that map `metric path -> tag`.  This cross reference table can be huge.

If in a Mysql world, then there will be a row for every tag, path combo.  In Cassandra we can take advantage of the
"list" type which is then `path -> <list>(tags)` and query that by `where <tag> in <list>`.

We also need to maintain a `tag name` index (so we can find what our tags acctually are), and a `tag name -> value` index.

Finally each metric has a "uniqueId" that is a hash of the `path + tags` so the real mapping of the unique number of metrics
is much bigger then just the simple "path" method.  If you have 1 million "path"s (in the old graphite world), and
start to add tags to them, you have effectively exponentially increased the number of unique metrics in the system by tag
permutations.

So what have we learned from this ::

    - If using tags, keep the `metric path` unique counts down to a nice small number
    - If not using tags, just keep on keepin' on


### Metrics2.0 + Carbon2

This is a "pure" tag format, there is no "key".  The closest thing to a "key" in this format is the "what" tag.
The metrics key is thus inferred from the tag list.  So basically this is a sort of redundant indexing mechanism, but it
allows us to use the usual graphite like "finders" for metrics.


## Triggered Rollups

For the mysql and cassandra series writers, there is an option to "trigger" rollups from the lowest resolution
rather then storing them in RAM.  Once the lowest resolution is "written" it will trigger a rollup event for the lower
resolution items which get written to the the DB.  This can solve 2 issues,

- No need for resolution caches (which if your metrics space is large, can be alot) and just needed for the highest resoltuion.

- Peristence delay: the Low resolutions can stay in ram for a long time before they are written.  This can be problematic if things crash/fail as
    those data points are never written.  With the triggered method, they are written once the highest resolution is written.

To enable this, simply use the writer drivers of

    cassandra-triggered or mysql-triggered or cassandra-log-triggered

to the `writer.metrics` options (it will be ignored for anything other then the mysql and cassandra blob writers)

### Modes

#### Burst mode `rollup_method="burst"`

This is the default method for non-log writers.
In this mode, rollups are computed once the initial low resolution write has competed.  For systems with alot
of unique metrics and a few higher resolutions, this can cause large spikes all the things (CPu, GC, RAM, etc) as rollups are
computed as fast as possible with little regard for the rest of what cadent might be doing.

I recommend this mode for a small amounts of unique metrics (~5-10k depending on your node size). Or if using the simple
overflow (i.e. not `log`) writer method as data inserts are of relatively low volume and constant over time.

You can tweak some settings to try to limit the impact.

    rollup_method="burst"
    rollup_workers=8  # more workers, more 
    rollup_queue_length=16384  # size before we start to block lowres inserts things
                

#### Queue mode `rollup_method="queue"`

For the `log` based writers this is the default mode.  Since the log method "dumps" the current chunk into the DB fast, 
this means alot of rollups are also triggered.  When dealing with many many unique metrics, this a) can take alot of time
and b) can hammer both the DB and Cadent (lots of getting, merging, writing going on).

There is also an issue with time.  Since rollups are expensive, it can take a fair amount of time to compute them all for
many unique keys.  If that time is greater then chunk flush time, we run into spiral of death.

The queue method queues a metric name with the earliest time, and slowly bleeds this queue.  The benefit of this method
is that the rollups are process more slowly (tunable of course), and won't effect the rest of cadent (i.e. the api, incoming, etc). 

If rollups don't complete by the time the next chunk is written, no worries, the last entry in the queue has an 
older time then the potential incoming, the newer one will not be added.  For alot of unique metrics, the rollup
process is a constant action.  The queue length is indeterminant in that it will hold as many as needed.

Some configuration is below (note: `rollup_queue_length` and `rollup_workers` are not relevant in this mode)

        rollup_method="queue"
        rollup_queue_workers=4  # default number of queue consumers
        



## Log

Unlike the other "series" based wrtiers that use the Size in ram before writing, this is pure time based.

It also is more tollerent of failures, by writing a temporary log also in the Database.

Every "N" seconds (default 10s) it will write a log of the current past N seconds of metrics to the DB
for a given sequence in a big-old-blob of zstd compressed json data.

Each sequence is "Y" seconds long (default 10min).  Since crashing or restarting cadent that has
many many thousands of metrics in ram caches that are not written, on restart this will re-read the last sequence
of the N second snapshots, and re-fill the caches (which if there are alot of metrics this to can take some time).

Once a time Y chunk is finished it is pushed to the front of the chunk list, and also written to the DB.

After the chunk is written it then removes the sequence slice logs from the DB.  Thus if things crash, restart
It only needs to grab the last chunk of data it had.

There can be "X" sequences held in ram for query speed (default of 6, plus the current one is 7 total).

### Writer options for log caches:

    cache_time_window = 10 # time in min for the chunk windows
    cache_num_chunks = 6 # number of chunks to be in ram (note there is a 7th chunk which is the current one)
    cache_log_flush_time = 10 # number of seconds for each log flush best of a multiple of the `accumulate_flush` window


### Performance notes:

While much nicer towards failures, it also can use more ram, and be a bit less efficent in terms
of space consumed by the series type (as they will be shorter typically then using straight byte sized limits).
Also since there can be more chunks to itterate over on queries, query speed will suffer a little.

One should also note there are more performance implications here.  Every "chunk" that gets writen can have 1000s ->
100000s+ series to flush.  They are flushed as fast as possible, and also rememeber that for each series that is
written, the rollups must also occur.

For example if there are 3 bins [10s, 1m, 10m] and ~1000 metrics in the caches as flush, for each metric written
2 up to 4 more writes will occur (as well as 2 reads) to fill the rollup series.  If the time it takes
to complete all those actions is longer then the `cache_time_window` one will end up in a race to the death.

As a concreate example, we have a system that consumes ~140k metrics in a 10m window, with 3 basic
bins.  It takes ~5-6min for the total flush to occur, using a 3 node Cassandra cluster, which is a bit close
to the `cache_time_window` (at least too close for me).  So we set things to have `cache_time_window=15` and
`cache_num_chunks=4` to have a bit more breathing room in case of more incoming metrics.

The acctual write time will vary alot based on your system configureation. So it's best to experiment a bit

One key metric emitted to watch, is to see how long

`stats.{statdprefix}.{host}.writer.rollup.process-time-ns.count`

takes.

Another thing to note on restarts: the log mechanism will attempt to slurp all the items in the logs table on startup.
Depending on the size and number of log chunks written this can take some time as many times it will need to slurp in
many millions of points.  While this happens, no "writes" are allow to occur to the cache mechanisms, however
injestion is still occuring.  Since the new points from the injestion are added to a queue to get written, there
can be a large RAM increase while both mechanisms start to fill in both the queue and the caches.  Once the caches
are up-to-date, writing begins as normal, and should drop the ram usage back to a stable level.
Given your system, it's best to have 2x the amount of RAM required for things for this to occur.  This is done for a
few reaons

1. We want to minimize the number of metrics that get lost due to no servers avilable.
2. Timeordering is important for certain series, so we cannot simply allow cache writes while the old logs are being injested.

Also the "API" is also allowed to run as well while this happens.  So any cached points not yet re-inserted will not
be available.  You may notice that certain graphs have the `cache_time_window` gap, and this will get filled
in while the log is read back in.

#### Cassandra Compaction:

##### Log Table:

Since each log write can contain alot of data, this log table will/can get large.  While each sequence does get purged
after a successful write of a the chunk.  Cassandra uses tombstones, not real deletes of the data. These tombstones
only get removed once a compation is run on that table. When tuning your cassandra cluster, this table should
get compacted more often then the others for this reason.

##### Triggered table-per-resolution:

The rollup process can also create alot of tobmstones due to the fact a rollup works as follows

1. select the last entry in the rollup table.
2. Merge that data with the new data from the smallest bin window we just wrote.
3. Upsert into the DB.

That 'upsert' in Cassandra is acctually a "delete" and "insert" (as data is treated as immutable) thus there will be
 more tombstones in those tables as well.


## Non-Log based Series writers: Max time in Cache

This applies only to Series/Blob writers where we store interal caches for a bunch of points. This does NOT
apply to the `Log` based wrtiers.

Since the default
behavior is to write the series only when it hits the `cache_byte_size` setting.  This can be problematic for series
that are very slow to update, and they will take a very long time to acctually persist to the DB system.  The setting

    max_time_in_cache

for the `writer.options` section lets you tune this value.  The default value is 1 hour (`max_time_in_cache=60m`)


### SubWriters

Currently there is support for "double writing" which simply means writing to 2 places at the same time (any more then
that and things can get abused at high volume).
Each writer gets it's own cache/queue/etc. Which means that the RAM requiements (and CPU) are doubled in the worst case
(it depends on the subwriter config of course).
This was mostly meant for writing to your long term storage, and publishing events to kafka.  Or as a migration path
for moving from one storage system to another.


### Shutdown

On shutdown (basically a SIGINT), since there can be lots of data still in the cache phase of things.  Cadent will attempt to Flush all the
data it has into the writers.  It this in "full blast" mode, which means it will simply write as fast as it can.  If
There are many thousands of metrics in RAM this can take some time (if you have multiple resolutions and/or in triggered
rollup mode things need to do these writes too so more time).   So expect loads on both the cadent host and the DB system
to suddenly go bonkers.  If you need to "stop" this action, you'll need a hard SIGKILL on the processes.

For the Log-based writers, this shutdown is quick as the log flushes have kept things in a state where upon
restart, the time is spent "pre-filling" the caches from these logs.

### Caches

The internal caches are the repos for all inflight data before it is written.  Some cache configurations are such that
writes only happen on an "overflow".  An overflow is when a meteric has reached some configurable "max size".  Any "series"
based writer uses this "overflow" meathod.  In this overflow meathod a "max time in cache" is also settable to force a write
for things that slow to get points added.

For Non-series writers, this overflow is set to "drop".  Drop means any NEW incoming points will not be added to the
write cache, we do this as w/o it there is a huge chance the ram requirements will OOM kill the process and more data
is then lost.  If you find yourself in this condition, we may need a bigger machine w/ more ram or faster DB system
for writing.

The API render pieces also need knowledge of these caches as in the "short" timescales (minuets-hours) most all data is
in ram.  So it also needs to know of the caches.  As a resul there is a config section just for caches for an accumulator section.
There can be many defined as if doing double writes (say to cassandra series format and whisper/kafka single stat emitters)
the caches mean different things as for series, we want it to do an overflow, where as in the single point we want to emit as
soon as possible.

Just a note the `kafka-flat` writer does not currently use a write back cache, as it's assumed to be able to handle the
incoming volume.  Obviously if your kafka cannot take the many hundreds of thousands of messages per second, i suggest the
series method is used.

#### Example

    [graphite-proxy-map]
    listen_server="graphite-proxy"
    default_backend="graphite-proxy"

        [graphite-proxy-map.accumulator]
        backend = "BLACKHOLE"
        input_format = "graphite"
        output_format = "graphite"
        random_ticker_start = false

        accumulate_flush = "5s"
        times = ["5s:1h", "1m:168h"]

        [[graphite-proxy-map.accumulator.writer.caches]]
            name="gorilla"
            series_encoding="gorilla"
            bytes_per_metric=1024
            max_metrics=1024000  # for 1 million points @ 1024 b you'll need lots o ram
            # max_time_in_cache="60m" # force a cache flush for metrics in ram longer then this number
            # broadcast_length="128" # buffer channel for pushed overflow writes
            # for non-series metrics, the typical behavior is to flush the highest counts first,
            # but that may mean lower counts never get written, this value "flips" the sorter at this % rate to
            # force the "smaller" ones to get written more often
            # low_fruit_rate= 0.25

        [[graphite-proxy-map.accumulator.writer.caches]]
            name="whisper"
            series_encoding="gob"
            bytes_per_metric=4096
            max_metrics=1024000
            # for non-series metrics, the typical behavior is to flush the highest counts first,
            # but that may mean lower counts never get written, this value "flips" the sorter at this % rate to
            # force the "smaller" ones to get written more often
            low_fruit_rate= 0.25

        [graphite.accumulator.writer.metrics]
            driver = "mysql-triggered"
            dsn = "user:pass@tcp(localhost:3306)/cadent"
            cache = "gorilla"

        [graphite.accumulator.writer.indexer]
            driver = "mysql"
            dsn = "user:pass@tcp(localhost:3306)/cadent"
            [graphite-proxy-map.accumulator.writer.indexer.options]
            writes_per_second=200

        # also push things to whisper files
        [graphite.accumulator.writer.submetrics]
            driver = "whisper"
            dsn = "/vol/graphite/storage/whisper/"
            cache = "whisper"

        [graphite.accumulator.writer.submetrics.options]
         xFilesFactor=0.3
         write_workers=16
         write_queue_length=102400
         writes_per_second=2500 # allowed physical writes per second


        # and a levelDB index
        [graphite.accumulator.writer.subindexer]
        driver = "leveldb"
        dsn = "/vol/graphite/storage/ldb/"

        [graphite.accumulator.api]
            base_path = "/graphite/"
            listen = "0.0.0.0:8085"
                [graphite-cassandra.accumulator.api.metrics]
                driver = "mysql-triggered"
                dsn = "user:pass@tcp(localhost:3306)/cadent"
                cache = "gorilla"

        [graphite.accumulator.api.indexer]
         driver = "leveldb"
         dsn = "/vol/graphite/storage/ldb/"


## Writer Schemas

## MySQL

### MYSQL-Flat + Index

Slap stuff in a MySQL DB .. not recommended for huge throughput, but maybe useful for some stuff ..
You should make Schemas like so (`datetime(6)` is microsecond resolution, if you only have second resolution on the
`times` probably best to keep that as "normal" `datetime`).  The TTLs are not relevant here.  The `path_table` is
useful for key space lookups

        CREATE TABLE `{segment_table}` (
            `segment` varchar(255) NOT NULL DEFAULT '',
            `pos` int NOT NULL,
            PRIMARY KEY (`pos`, `segment`)
        );

        CREATE TABLE `{path_table}` (
            `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
            `segment` varchar(255) NOT NULL,
            `pos` int NOT NULL,
            `uid` varchar(50) NOT NULL,
            `path` varchar(255) NOT NULL DEFAULT '',
            `length` int NOT NULL,
            `has_data` bool DEFAULT 0,
            PRIMARY KEY (`id`),
            KEY `seg_pos` (`segment`, `pos`),
            KEY `uid` (`uid`),
            KEY `length` (`length`)
        );


    CREATE TABLE `{tag_table}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `name` varchar(255) NOT NULL,
      `value` varchar(255) NOT NULL,
      `is_meta` tinyint(1) NOT NULL DEFAULT 0,
      PRIMARY KEY (`id`),
      KEY `name` (`name`),
      UNIQUE KEY `uid` (`value`, `name`, `is_meta`)
    );

    CREATE TABLE `{tag_table}_xref` (
      `tag_id` BIGINT unsigned,
      `uid` varchar(50) NOT NULL,
      PRIMARY KEY (`tag_id`, `uid`)
    );


    CREATE TABLE {table}_{resolution}s (
          id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
          uid varchar(50) CHARACTER SET ascii NOT NULL,
          path varchar(255) NOT NULL DEFAULT '',
          sum float NULL,
          min float NULL,
          max float NULL,
          last float NULL,
          count float NULL,
          time datetime(6) NOT NULL,
          PRIMARY KEY (id),
          KEY uid (uid),
          KEY time (time)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

If your for times are `times = ["10s", "1m", "10m"]` you should make 3 tables named

    {tablebase}_10s
    {tablebase}_60s
    {tablebase}_600s

And only ONE path/tag tables

    {path_table}, {tag_table}, and {tag_table_xref}

Config Options

    # Mysql
    #  NOTE: this expects {table}_{keepertimesinseconds} tables existing
    #  if timers = ["5s", "10s", "1m"]
    #  tables "{table}_5s", "{table}_10s" and "{table}_60s"
    #  must exist
    [mypregename.accumulator.writer]
    driver = "mysql"
    dsn = "root:password@tcp(localhost:3306)/test"
        [mypregename.accumulator.writer.options]
        table = "metrics"
        path_table = "metrics_path"
        tag_table = "metrics_tag"
        tag_table_xref = "metrics_tag_xref"
        tags = "host=localhost,env=dev" # static tags to include w/ every metric

        batch_count = 1000  # batch up this amount for inserts (faster then single line by line) (default 1000)
        periodic_flush= "1s" # regardless if batch_count met, always flush things at this interval (default 1s)


### MYSQL - blob

The index table the same if using mysql for that.  The Blob table is different of course.  Unlike the flat writer
which may have to contend with many thousands of writes/sec, this one does not have a write cache buffer as writes
should be much "less".  There will be times of course when many series need to be written and hit their byte limit
If this becomes an issue while testing, the write-queue mechanism will be re-instated.

    CREATE TABLE `{table}_{resolution}s` (
      `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) DEFAULT CHARACTER SET ascii NOT NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `ptype` tinyint(4) NOT NULL,
      `points` blob,
      `stime` bigint(20) unsigned NOT NULL,
      `etime` bigint(20) unsigned NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`, `etime`),
      KEY `path` (`path`)
    ) ENGINE=InnoDB;


The actual "get" query is `select point_type, points where uid={uiqueid} and etime >= {starttime} and etime <= {endtime}`.
So the index of (uid, etime) is proper.


Config Options

    [[mypregename.accumulator.caches]]
    name="gorilla-sql"
    series_encoding="gorilla"
    bytes_per_metric=1024
    max_metrics=1024000

    [mypregename.accumulator.writer]
    driver = "mysql"
    dsn = "$ENV{MYSQL_USER:admin}:$ENV{MYSQL_PASS:}@tcp(localhost:3306)/cadent"
        [mypregename.accumulator.writer.options]
        table = "metrics"
        path_table = "metrics_path"
        cache = "gorilla-sql"
        tags = "host=localhost,env=dev" # static tags to include w/ every metric
        expire_on_ttl = true        # this will run a periodic job to purge "TTL'ed" data in all the tables of metrics



## File

Good for just testing stuff or, well, other random inputs not yet supported
This will dump a TAB delimited file per `times` item of

    statname uniqueid sum min max last count resolution nano_timestamp nano_ttl

If your for times are `times = ["10s", "1m", "10m"]` you will get 3 files of the names.

    {filebase}_10s
    {filebase}_60s
    {filebase}_600s


Config Options

    # File
    #  if [keepers] timers = ["5s", "10s", "1m"]
    #  files "{filename}_5s", "{filename}_10s" and "{filename}_60s"
    #  will be created
    #
    # this will also AutoRotate files after a certain size is reached
    [mypregename.accumulator.writer]
    driver = "file"
    dsn = "/path/to/my/filename"
        [mypregename.accumulator.writer.options]
        max_file_size = "104857600"  # max size in bytes of the before rotated (default 100Mb = 104857600)

## Cassandra

NOTE: there are 2 cassandra "modes" .. Flat and Blob

Flat: store every "time, min, max, sun, count, last" in a single row
Blob: store a "chunk" of a byte size (16kb default) in a bit packed compressed blob (a "TimeSeries")

Regardless of choice ...

This is probably the best one for massive stat volume. It expects the schema like the MySQL version,
and you should certainly use 2.2 versions of cassandra.  Unlike the others, due to Cassandra's type goodness
there is no need to make "tables per timer".  Expiration of data is up to you to define in your global TTLs for the schemas.
This is modeled after the `Cyanite` (http://cyanite.io/) schema as the rest of the graphite API can probably be
used using the helper tools that ecosystem provides.  (https://github.com/pyr/cyanite/blob/master/doc/schema.cql).
There is one large difference between this and Cyanite, the metrics point contains the "count" which is different
then Cyanite as they group their metrics by "path + resolution + precision", i think this is due to the fact they
dont' assume a consistent hashing frontend (and so multiple servers can insert the same metric for the same time frame
but the one with the "most" counts wins in aggregation) .. but then my Clojure skills = 0.
For consistent hashing of keys, this should not happen.

If at all possible use Cassandra >= 3.0.9 as has the new `TimeWindowCompactionStrategy` which supperceeds the
`DateTieredCompactionStrategy`.

Please note for now the system assumes there is a `.` naming for metrics names

    my.metric.is.fun


You should wield some Cassandra knowledge to change the on the `metric.metric` table based on your needs
The below causes most compaction activity to occur at 10m (min_threshold * base_time_seconds)
and 2h (`max_sstable_age_days` * `SecondsPerDay`) windows.
If you want to allow 24h windows, simply raise `max_sstable_age_days` to ‘1.0’.

    compaction = {
        'class': 'DateTieredCompactionStrategy',
        'min_threshold': '12',
        'max_threshold': '32',
        'max_sstable_age_days': '1',
        'base_time_seconds': '50'
    }
    
Things attempt to discover the cassandra version.  So if things are in the >3 range we use this instead

    compaction = {
        'class': 'TimeWindowCompactionStrategy',
        'compaction_window_unit': 'DAYS',
        'compaction_window_size': '1'
    }


### Cassandra Flat Schema

    CREATE KEYSPACE metric WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '3'}  AND durable_writes = true;

    USE metric;

    CREATE TYPE metric_point (
            mx double,
            mi double,
            s double,
            l double,
            c int
        );


        CREATE TYPE metric_id_res (
            id varchar,
            res int
        );

        CREATE TABLE metric.metric (
            mid frozen<metric_id_res>,
            t bigint,
            pt frozen<metric_point>,
            PRIMARY KEY (id, t)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (mid ASC, t ASC)
            AND compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '0.083',
                'base_time_seconds': '50'
            }
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

        CREATE TYPE metric.segment_pos (
            pos int,
            segment text
        );


        CREATE TABLE metric.path (
            segment frozen<segment_pos>,
            length int,
            path text,
            id varchar,
            has_data boolean,
            PRIMARY KEY (segment, length, path, id)
        ) WITH compaction = {'class': 'org.apache.cassandra.db.compaction.SizeTieredCompactionStrategy'}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

        CREATE INDEX ON metric.path (id);

        CREATE TABLE metric.segment (
            pos int,
            segment text,
            PRIMARY KEY (pos, segment)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (segment ASC)
            AND compaction = {'class': 'org.apache.cassandra.db.compaction.SizeTieredCompactionStrategy'}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

### Blob Schema

Much the same, but instead we store the bytes blob of the series.  `ptype` is the encoding of the blob itself.
Since different resolutions in cassandra are stored in one super table, we need to disinguish the id+resolution
 as a unique id.

        CREATE TABLE metric.metric (
            mid frozen<metric_id_res>,
            etime bigint,
            stime bigint,
            ptype int,
            points blob,
            PRIMARY KEY (mid, etime)
        ) WITH CLUSTERING ORDER BY etime ASC
            AND compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '0.083',
                'base_time_seconds': '50',
                'tombstone_threshold': 0.05
            }
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

### FlatMap Schema

This is really a flat list as it turns out to be a) allow same times in the list (for the BYPASS mode) and is 
a bit faster then using maps.

    CREATE TYPE IF NOT EXISTS metric.metric_point (
        t bigint,
        mx double,
        mi double,
        s double,
        l double,
        c int
    );

    CREATE TYPE IF NOT EXISTS   metric.metric_id_res (
        id ascii,
        res int
    );

    CREATE TABLE IF NOT EXISTS  metric.metrics (
        uid ascii,
        res int,
        slab ascii,
        ord ascii,
        pts list<frozen<metric_set_point>>,
        PRIMARY KEY ((uid, res, slab), ord)
    ) WITH CLUSTERING ORDER BY (ord ASC) AND
   
    compaction = {
        'class': 'TimeWindowCompactionStrategy',
        'compaction_window_unit': 'DAYS',
        'compaction_window_size': '1'
    }
    AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

For "table_per_res" mode, we alter things to

    CREATE TABLE IF NOT EXISTS  metric.metrics_{{resolution}}s (
        uid ascii,
        slab ascii,
        ord ascii,
        pts list<frozen<metric_set_point>>,
        PRIMARY KEY ((uid, slab), ord)
    ) WITH CLUSTERING ORDER BY (ord ASC) AND
   
    compaction = {
        'class': 'TimeWindowCompactionStrategy',
        'compaction_window_unit': 'DAYS',
        'compaction_window_size': '1'
    }
    AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};



### NOTE: for Cassandra 3

There's a better compaction method for the metrics that have pretty much constant time inputs (i.e. server metrics)
so i recommend doing below.


        CREATE TABLE metric.metric (
            mid frozen<metric_id_res>,
            etime bigint,
            stime bigint,
            ptype int,
            points blob,
            PRIMARY KEY (mid, etime)
        ) WITH CLUSTERING ORDER BY etime ASC
        AND compaction = {
            'class': 'TimeWindowCompactionStrategy',
            'compaction_window_unit': 'DAYS',
            'timestamp_resolution': 'SECONDS',
            'compaction_window_size': '1',
            'tombstone_threshold': 0.05
        }
        AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
        AND read_repair_chance = 0,
        AND dclocal_read_repair_chance = 0;


     -- Or for single metric items


       CREATE TABLE metric.metric (
           mid frozen<metric_id_res>,
           time bigint,
           point frozen<metric_point>,
           PRIMARY KEY (id, time)
       ) WITH COMPACT STORAGE
           AND CLUSTERING ORDER BY (mid ASC, time ASC)
           AND compaction = {
               'class': 'TimeWindowCompactionStrategy',
               'compaction_window_unit': 'DAYS',
               'timestamp_resolution': 'SECONDS',
               'compaction_window_size': '1',
                'tombstone_threshold': 0.05
           }
           AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'},
           AND read_repair_chance = 0,
           AND dclocal_read_repair_chance = 0;


DO NOT use this if things will be inserted "out of order by time" data or have alot of sparse data.

So you may wish to change the `compaction_window_size` to suit your query/insert patterns. Also change the
`timestamp_resolution` to the acctuall resolution you need.  Keep in mind that Cadent assumes second resolution for
any queries, as that's what graphite does/did.

For instance a 3 DAY size w/ expireing TTLs of 90 DAYS is good.  Or a 1 DAY size w/ 30 DAYS and so on.


### Cassandra + Table Per Resolution

If in the Options for the writes you specify `table_per_resolution` then we adopt the same model that we do for MySQL.

Things expect a table names `{metric_table}_{resolution in seconds}s` for each resolution. Like so

    metric_1s
    metric_10s
    metric_60s
    etc

This is acctually a good way to handle cassandra effectively.  Since in a trigger rollup world, we must "delete and re-insert"
the row for the rollup rows (or in this case tables).  Those tables will have many tombstones and need more compaction,
but since their resolution is smaller there will much less data to rollup.  This also lets you use the
`TimeWindowCompactionStrategy` more effecively in Cassandra 3, in that you can specifiy `compaction_window_size` more
appropriate for your TTL on the data.  It also keeps the write/read volume for the "quick (highest resolution)" data
out of the picture for doing rollups (if using triggered rollups), thus making things more effcient there.

The potential only issue is that you're not able to "change resolutions", but if this needs to happen, you're better
off "restarting" everything anyway as all the old data is going to be hard to query and match up.

    [myaccumulator.accumulator.writers.metrics.options]
    table_per_resolution=true

Remember you should add your tables to use the proper `compaction_window_size` you need.

* REMEMBER * to set the same option in the API section (otherwise it will look for the know what to look for)

        [myaccumulator.accumulator.api.metrics.options]
        table_per_resolution=true

The schema is different as well since we no longer need a "resolution" in the data point

        CREATE TABLE metric.metric_{resolution}s (
            id ascii,
            etime bigint,
            stime bigint,
            ptype int,
            points blob,
            PRIMARY KEY (mid, etime)
        ) WITH CLUSTERING ORDER BY etime ASC


### Cassandra + Gotcha's

Some notes from the field::

Write Speed:: Cassandra Protocol v3 (cassandra <= 2.1) is MUCH slower then Protocol v4 (cassandra 2.2 -> 3.X).

Given that we may to need to write ~100-200 THOUSANDs metric points in our flush window (typically 5s-10s)
if we cannot fully write all the stats in the flush window beteen flush times, the app will have to SKIP a flush write
in order to basically not die a horrible death of RAM consumption and deadlocks.

As a result .. don't use cassandra 2.1, use at least cassandra 2.2

The tickers for the flushes attempt to try to flush things on proper mod internals
The ticker will try to start on `time % duration == 0` this is not exact, but it usually hits within a few milliseconds of the correct mode.
i.e. a "timer" of "10s" should start on `14579895[0-9]0`

To further make Cassandra data points and timers align, FLUSH times should all be the Smallest timer and timers should be multiples of each other
(this is not a rule, but you really should do it if trying to imitate graphite whisper data), an example config below

    [graphite-cassandra]
    listen_server="graphite-proxy"
    default_backend="graphite-proxy"

    # accumulator and
    [graphite-cassandra.accumulator]
    backend = "graphite-gg-relay"
    input_format = "graphite"
    output_format = "graphite"

    # push out to writer aggregator collector and the backend every 10s
    # this should match the FIRST time in your times below, but is not totally nessesary
    accumulate_flush = "10s"

    # aggregate bin counts
    times = ["10s:168h", "1m:720h", "10m:21600h"]

    [graphite-cassandra.accumulator.writer.metrics]

## ElasticSearch

Probably makes the most sence for Indexing data efficently, however, the metrics can also be passed to this backend store

### Indexing


This is the Index schema for elastic search

    metric_path/path mapping
    {
        "properties":{
            "uid":{
                "type": "string",
                "index": "not_analyzed"
            },
            "segment":{
                "type": "string",
                "index": "not_analyzed"
            },
            "path":{
                "type": "string",
                "index": "not_analyzed"
            },
            "pos": {
                "type": "integer",
                "index": "not_analyzed"
            },
            "length": {
                "type": "integer",
                "index": "not_analyzed"
            },
            "has_data": {
                "type": "boolean",
                "index":  "not_analyzed"
            },
            "tags":{
                "type": "nested",
                "properties":{
                    "name": {
                        "type": "string",
                        "index": "not_analyzed"
                    },
                    "value": {
                        "type": "string",
                        "index": "not_analyzed"
                    },
                    "is_meta":{
                        "type":"boolean",
                        "index": "not_analyzed"
                    }
                }
            }
        }
    }

    metric_segment/segment mapping
    {
        "properties":{
           "segment":{
                "type": "string",
                "index": "not_analyzed"
            },
            "pos": {
                "type": "integer",
                "index": "not_analyzed"
            }
        }
    }

    metric_tag/tag mapping
    {
        "properties": {
            "name":{
                "type": "string",
                "index": "not_analyzed"
            },
            "value": {
                "type": "string",
                "index":  "not_analyzed"
            },
            "is_meta": {
                "type": "boolean",
                "index": "not_analyzed"
            }
        }
    }

### Metrics - Flat


The `_id` for these entries is {uid}-{nanosecond-timestamp} of the incoming metric, so it is possible to overwrite
metrics here.

Also note, that instead of a write-back cache, we instead do Batch inserts.  The default is 1000 metrics per batch.
Given there can be many batches for "high" volumes you may wish to tweak the thread/queue size for elastic search
in the .yml config file, otherwise inserts can fail.  If some in a batch to fail to get inserted they will be re-added
to the batch queue to be attempted again.

    threadpool:
        bulk:
            queue_size: 5000


Below is the basic schema for the indexed metrics.  Please note that I would only use ES for metrics storage if
you have out-of-order incoming metrics per key and their relative volume is "small" (small is subjective, but
since the system may need to inject many thousands of metric points per second, performance matters).  The storage
format for ES is also not the most efficient and takes up alot of disk space for a relatively small volme of metrics.

    {
       "dynamic_templates": [{
                "notanalyze": {
                    "mapping": {
                            "index": "not_analyzed",
                            "omit_norms": true
                    },
                    "match_mapping_type": "*",
                    "match": "*"
                }
       }],
       "_all": {
        "enabled": false
       },
       "properties":{
            "uid":{
                "type": "string",
                "index": "not_analyzed"
            },
            "path":{
                "type": "string",
                "index": "not_analyzed"
            },
            "time":{
                "type": "date",
                "index": "not_analyzed",
                "format": "strict_date_optional_time||epoch_millis"
            },
            "min":{
                "type": "double",
                "index": "not_analyzed"
            },
            "max":{
                "type": "double",
                "index": "not_analyzed"
            },
            "sum":{
                "type": "double",
                "index": "not_analyzed"
            },
            "last":{
                "type": "double",
                "index": "not_analyzed"
            },
            "count":{
                "type": "long",
                "index": "not_analyzed"
            },
            "tags":{
                "type": "nested",
                "properties":{
                    "name": {
                        "type": "string",
                        "index": "not_analyzed"
                    },
                    "value": {
                        "type": "string",
                        "index": "not_analyzed"
                    },
                    "is_meta":{
                        "type":"boolean",
                        "index": "not_analyzed"
                    }
                }
            }
       }
    }

An example config is below


    # cache objects to be shared (or not) across the writer backends
    # even though it is not used (yet) for ES one is still required.
    [[graphite-proxy-map.accumulator.writer.caches]]
    name="dummy"
    series_encoding="gob"

    [graphite-proxy-map.accumulator.writer.metrics]
        driver = "elasticsearch-flat"
        dsn = "http://127.0.0.1:9200"
        cache = "dummy"

        [graphite-proxy-map.accumulator.writer.metrics.options]
        batch_count=1000  # batch size for inserts
        metric_index = "metrics_flat"


    [graphite-proxy-map.accumulator.writer.indexer]
        driver = "elasticsearch"
        dsn = "http://127.0.0.1:9200"
        [graphite-proxy-map.accumulator.writer.indexer.options]
        writes_per_second=100

    [graphite-proxy-map.accumulator.api]
            base_path = "/graphite/"
            listen = "0.0.0.0:8083"

            [graphite-proxy-map.accumulator.api.metrics]
                driver = "elasticsearch-flat"
                dsn = "http://127.0.0.1:9200"
                cache = "gob"
                [graphite-proxy-map.accumulator.api.metrics.options]
                metric_index="metrics_flat"

            [graphite-proxy-map.accumulator.api.indexer]
                driver = "elasticsearch"
                dsn = "http://127.0.0.1:9200"
## Whisper

Yep, we can even act like good old carbon-cache.py (not exactly, but close).  If you want to write some whisper files
that can be used choose the whisper writer driver.  Unlink the carbon-cache, here only "one set" of aggregate timers
is allowed (just the `times=[...]` field) for all the metrics written (i.e. there is no `storage-schema.cof` built in yet).
Also this module will attempt to "infer" the aggregation method based on the metric name (sum, mean, min, max) rather
then using `storage-aggregation.con` for now.

Aggregation "guesses"  (these aggregation method also apply to the column chosen in the Cassandra/Mysql drivers)

        endsin "mean":              "mean"
        endsin "avg":               "mean"
        endsin "average":           "mean"
        endsin "count":             "sum"
        startswith "stats.count":   "sum"
        endsin "errors":            "sum"
        endsin "error":             "sum"
        endsin "requests":          "sum"
        endsin "max":               "max"
        endsin "min":               "min"
        endsin "upper_\d+":         "max"
        endsin "upper":             "max"
        endsin "lower_\d+":         "min"
        endsin "lower":             "min"
        startswith "stats.gauge":   "last"
        endsin "gauge":             "last"
        endsin "std":               "mean"
        default:                    "mean"


An example config below


    [graphite-whisper]
    listen_server="graphite-proxy"
    default_backend="graphite-proxy"

     # accumulator and
     [graphite-whisper.accumulator]
        backend = "BLACKHOLE"
        input_format = "graphite"
        output_format = "graphite"

     # push out to writer aggregator collector and the backend every 10s
     # this should match the FIRST time in your times below
     accumulate_flush = "10s"

     # aggregate bin counts
     times = ["10s:168h", "1m:720h", "10m:21600h"]

     [graphite-whisper.accumulator.writer.metrics]
        driver="whisper"
     	dsn="/root/metrics/path"
     	xFilesFactor=0.3
     	write_workers=32
     	write_queue_length=102400

     [graphite-whisper.accumulator.writer.indexer]
        driver="whisper"
        dsn="/root/metrics/path"


## KAFKA + Kafka-Flat

I mean why not.  There is no "reader" API available for this mode, as kafka it's not designed to be that.  But you can
shuffle your stats to the kafka bus if needed.  There are 2 message types "index" and "metrics".  They can be
put on the same topic or each in a different one, the choice is yours.  Below is the 2 messages JSON blobs.
You can set `write_index = false` if you want to NOT write the index message (as the metric message has the metric in it
already and consumers can deal with indexing)


        INDEX {
            id: uint64 FNVa,
            uid: base36(id),
    	    type: "index | delete-index",
    	    path: "my.metric.is.good",
    	    segments: ["my", "metric", "is", "good"],
    	    senttime: [int64 unix Nano second time stamp]
    	    tags: []Tag //[{name: xx, value: xx }, {name: xx, value: xx}]
            meta_tags: []Tag // [{name: xx, value: xx }, {name: xx, value: xx}]
    	}

The "Flat" format is

    	METRIC{
    	    single: {
    	        time: [int64 unix Nano second time stamp],
    	        metric: "my.metric.is.good",
    	        id: uint64 FNVa,
    	        uid: string // base 36 from the ID
    	        sum: float64,
    	        min: float64,
    	        max: float64,
    	        last: float64,
    	        count: int64,
    	        resolution: float64,
    	        ttl: int64
                tags: [ {name: xx, value: xx }, {name: xx, value: xx}]
                meta_tags:  [ {name: xx, value: xx }, {name: xx, value: xx}]
    	    }
    	}

The "Blob" format is

    	METRIC{
    	    series: {
    	        time: [int64 unix Nano second time stamp],
    	        metric: "my.metric.is.good",
    	        id: uint64 FNVa,
    	        uid: string // base 36 from the ID
    	        data: bytes,
    	        encoding: string // the series encoding gorilla, protobuf, etc
    	        resolution: float64,
    	        ttl: int64,
    	        tags:  [ {name: xx, value: xx }, {name: xx, value: xx} ...]
                meta_tags:  [ {name: xx, value: xx }, {name: xx, value: xx} ...]
            }
    	}

Encoding formats supported are "json", "protobuf" and "msgpack"

Where as the "flat" format is basically a stream of incoming accumulated values, the blob format is

Here are the configuration options

            [[to-kafka.accumulator.writer.caches]]
                    name="gorilla-kafak"
                    series_encoding="gorilla"
                    bytes_per_metric=1024
                    max_metrics=1024000

            [to-kafka.accumulator.writer.metrics]
            driver = "kafka" // or "kafka-flat"
            dsn = "pathtokafka:9092,pathtokafka2:9092"
            cache="gorilla-kafka" # number of metrics to aggrigate before we must drop
            [to-kafka.accumulator.writer.metrics.options]
        	    # some kafka options
        	    compress = "snappy|gzip|none" (default: none)
        	    max_retry = 10
        	    ack_type = "local" # (all = all replicas ack, default "local")
        	    flush_time = "1s" # flush produced messages ever tick (default "1s")
        	    tags = "host=host1,env=prod" # these are static for whatever process is running this
        	    index_topic = "cadent" # topic for index message (default: cadent)
                metric_topic = "cadent" # topic for data messages (default: cadent)
                encoding = "msgpack"  # can be "json" or "msgpack" or "protobuf"


        	[to-kafka..accumulator.writer.indexer]
            driver = "kafka"
            dsn = "pathtokafka:9092,pathtokafka2:9092"

        	    [to-kafka..accumulator.writer.indexer.options]
                write_index = false|true

If you want to bypass the entire "graphite" thing and go straight to a kafka dump, look to
`configs/statsd-kafka-config.toml` and `configs/statsd-kafka-prereg.toml` pair .. this is probably the most typical use
of a kafka writer backend.  One can easily do the same with straight graphite data (i.e. from `diamond` or similar).


Since ordering and time lag and all sorts of other things can mess w/ the works for things, it's still best to fire stats to
a consistent hash location, that properly routes and aggregates keys to times such that stats are emitted "once" at a given time
window.  In that way, ordering/time differences are avoided.  Basically  `statsd -> flush to consthasher -> route -> flush to kafka`



## Whisper

99% of the performance issue w/ Wisper files are the Disks.  Since we typically here have large space requirements
(in the TB range) and we are in the Cloud (AWS for us).  We need to use EBS drives which are really slow compared
to any SSDs in the mix.  So you MUST limit the `writes_per_second` allowed or you'll end up in IOWait death.  For a
1 Tb (3000 iops) generic EBS (gp2) drive empirically we find that we get ~1500 batch point writes per/second max
(that's using all the IOPs available, which if you think of each "write" as needing to read first then write that
makes some sense).  So we set the `writes_per_second=1200` to allow for readers to actually function a bit.


## Some notes about RAM  
### Writers Cache/Ram

This one is a bit tricky to figure out exactly, and it highly dependent on the metric volume and shortest "tick" interval.
The cache ram needed depends on # metrics/keys and the write capacity.  The cache holds a `map[key][]points`.  Once
the writer determines which metric to flush to disk/DB we reclaim the RAM.

Just some empirical numbers to gauge things, but the metric you should "watch" about the ram consumed by the cache are
found in `stats.gauges.{statsd_prefix}.{hostname}.cacher.{bytes|metrics|points}`.

    Specs:
        Instance: c4.xlarge
        EBS drive: 1TB (3000 IOPS)
        Flush Window: 10s
        Keys Incoming: 140,000
        Writes/s: 1200(set) ~1000 (actual)
        CacheRam consumed: ~300-600MB
        # Points in Cache: ~1.3 Million

The process that runs the graphite writer then consumes ~1.2GB of ram in total.  Assuming the key space does not
increase (by "alot") the above is pretty much a steady state.

#### "Flat" writers

This is the easiest to figure out capacity in terms of ram.  Basically while things are being aggregated you will need
at least `N * (8*8 bytes + key/tag lengths)` (N metrics) As each metric+tag set is in Ram until it hits a flush window
For example.  If you have 10000 metrics, you will need roughly ~200Mb of ram, JUST for the aggregation phase
FOR EACH RESOLUTION.  So if you have 5s, 1m, 10m flush windows you will need ~600-700Mb.

Since the "flat" metrics are flushed to write-back buffers, each flush will end up copying that aggregation buffer into
another set of lists.  For each point that's flushed, and not written yet, double your ram requirements.  Depending
on the speed of the writers, this can get pretty large.  For slow writers this can add up, so keep that in mind.

For things like redis/cassandra, where writes are very fast, these buffers will be much smaller.

The Elasticsearch backend currently does use "caching" so they attempt to write as fast as they can (kindof an experiment).  So you'll
 know pretty fast if your backend needs expanding.  Using ES for metrics is a-ok so long as your volume is not in the
 many 10-100k/second.  ES can be expanded for that kind of volume, but it's not cheap.

Kafka also does not use caching for it writing as like cassandra "writes" are pretty fast and it's batching is also as it
does not need to index or compute much on handoff.

I've been using statblast + cadent to really experiment with different DBs and their "edge" capabilities and optimize around them. 
Given the data randomness and volume for most large systems.  If you want to stress test something, there's nothing like 
trying to write many 10k things a second.


#### "Blob" writers

The timeseries binary blobs are a bit harder to figure out in therms of their exact RAM requirements as some of them
have variable compression based on the in values themselves (the Gorilla/Gob series for instance).  Others like
Protobuf/Msgpack have pretty predictable sizes, but they too can use variable bit encodings for things so it's not
written in stone.  And unlike the flat writers, series are only writen when they fill their `bytes_per_metric` or
`cache_time_window` (if using the log-chunk method) setting.

But a "worse case" is easily determined as:

`NumResolutions * MaxNumMetrics * 7*64 (7 64bit nums) * 255 (worst case metric name size)`

That said, the Blob writers will "write for real" when they hit their configured byte threshold. So for an 8kb threshold

`NumResolutions * MaxNumMetrics * 8kb * 255 (worst case metric name size) = TotalRam Consumed` plus the above RAM
needed for just keeping the current set of aggregates. (And of course there is overhead associated with everything so give at
least 25% on top of that).

The difference is that all that data is stored in RAM and the qurey engine knows not to even bother with the backend
storage for the data, so read queries for hot/recent data are very fast.

Random experimentation using the Gorrilla "wide" format (where it needs to store all 8 64 numbers), 120k metrics w/ 2 resolutions
at 8kb block size is about 3Gb-4Gb for everything.

For those in AWS.  The r3 series is your friend or a big c3/4 as CPU cycles to ingest all the incoming is also important.
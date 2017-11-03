
Injectors
---------

Injectors are an "interface" to external non-tcp socket based feeds.  These injectors "skip" the accumulator phase
and are direct writers (like the ByPass in the accumulators but w/o any other frontend feeders).  

Currently "kafka" is the only injector supported, and kafka 0.10.1 at that. So we will focus on just that setup.

*NOTE: currently only the `cassandra-triggered` metric writer backend is supported (any indexer can be used).*

# Setup

## Kafka

You will need two topics (they are not automatically created).  The first topic is for your metrics.  This should be a normal topic type.

    /usr/share/kafka/bin/kafka-topics.sh --zookeeper={ZOOKEEPER} --create --topic=cadent \
    --replication=3 --partitions=XXX 

The second is for the write commits.  This one, since it is an offset tracker per metric key, should be a "compact" version, and should
have the same number of partitions the data topic does.

    /usr/share/kafka/bin/kafka-topics.sh --zookeeper={ZOOKEEPER} --create \
    --topic=cadent-commits --replication=3 --partitions=XXX --config cleanup.policy=compact \
    --config min.cleanable.dirty.ratio=0.3 -config delete.retention.ms=60000

## Getting Metrics Into Kafka

Since this is a bypass mode, you probably do want to inject statsd/graphite data directly into the stream as you'll
explode the DBs, so it's best to have the "accumulator + kafka" writer emitting stats into the topics at hand

    ./cadent --config=configs/graphite/graphite-config.toml --prereg=configs/graphite/graphite-kafka-flat.toml

This will inject a `SingleMetric` encoding in msgpack (probably the best performance encoder/decoder that includes a schema
protobufs probably would be faster, however it is a schema-less protocol).

See more about the supported schemas look here https://github.com/wyndhblb/cadent/blob/master/src/cadent/server/schemas/series/series.proto.

A single metric is of the form

    message SingleMetric {
        uint64 id = 2;
        string uid = 3;
        int64 time = 4;
        string metric = 5;
        double min = 6;
        double max = 7;
        double last = 8;
        double sum = 9;
        int64 count =10;
        uint32 resolution = 11;
        uint32 ttl = 12;
        repeated repr.Tag tags = 13;
        repeated repr.Tag meta_tags = 14;
        int64 sent_time = 15;
    }

And is probably the best format for metric injection.

### Reading out from kafka

To fire up the consumer

    ./cadent --injector=configs/injector/kafka-cassandra-series-pertable.toml 
 

## Considerations + Partition IDs

Kafka partitions things via some "hash" on a "key" for us that key is ALWAYS `UniqueIdString` == `base36(fnv64a(key + ":" +tags))`

This is very important as in order to keep tabs on where we've written, we need another topic the `commit` topic that keeps
track of the various `UniqueIdStrings` and the last offset we actually wrote to the database.

This turns out to be a very asynchronous kind of kafka streams where, unlike kafka or spark streams, the `windowing` functions
is not a fixed in time or fixed in space process.  Thus a metric series can be written in any order regardless of the offset 
kafka thinks our consumer is on.

For this reason the Topics Partitions count should be carefully chosen to match your assumed volume (more is better then few)
and ALL topics must match in partition counts.

If you're expecting millions of metrics (keys, not distinct values) every minuet, 128, 256 partitions is good.

In this way we have a one-to-one mapping between "data partitions" and "commit partitions"

Consumers need to be assigned their Partition IDs in the config (the default is "all" but that's just for small installs).


### NOTE:

Something to keep in mind is that this will *NOT* be a rebalanced type of configuration (where here can be a mechanism for consumers
to attach to given partitions if they are not claimed, or a node fails) .. Should one of the nodes fail, you'll need to bring another node 
up and let it re-sync from where the old consumer(s) left off on the same partitions.


### Partition assignment 

To assign partitions simple set the `consume_partitions` setting

    consume_partitions="all"
    consume_partitions="0,10,20,30"

This will hook the consumers for both the commit log and data consumers into the same set of partitions.


## Stages

On startup, cadent will consume all the keys from the commit log and make an internal map of the latest offsets written 
for each key.  After that finishes the data partitions are read from the minimum offset given by the commit logs. Any offsets
already consumed/written for a given metric will be skipped.

Once the series cache is "full" or "expired" (depending on your settings) if gets flushed to the backend main DB, and then 
a commit message is sent.

## A little data race

So yes there can be a condition where the DB was flushed and the commit message was never written (crash or other mean thing) .. in that case
the series may be overwritten assuming it has the same start and endtimes on a subsequent flush operation, but that should be ok. In this case
The rollup mechanism may double count that.


## A little setup 

![A Kafka of Flow](./kafka-flow.png)

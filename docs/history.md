
# How Cadent was started ...

Once upon a time, our statsd servers had __waaaayyyy__ to many UDP errors (a common problem I've been told).
Like many projects, necessity, breeds, (well, more necessity, but that's another conversation)...
So a Statsd Consistent Hashing Proxy in Golang was written to be alot faster and not drop any packets.

And then ...

This was designed to be an arbitrary proxy for any "line" that has a representative Key that needs to be forwarded to
another server consistently (think carbon-relay.py in graphite and proxy.js in statsd) but it's able to handle
any other "line" that can be regex out a Key, like fowarding loglines to an ELK cluster farm.

And then ...

It Supports health checking to remove nodes from the hash ring should the go out, and uses a pooling for
outgoing connections (mostly because the cloud is a nice place).

It also supports various hashing algorithms to attempt to imitate other
proxies to be transparent to them.

And then ...

Replication is supported as well to other places, there is "alot" of pooling and line buffereing so we don't
waste sockets and time sending one line at a time for things that can support multiline inputs (i.e. statsd/graphite)

And then ...

A "Regex" filter on all incoming lines to either shift them to other backends (defined in the config) or
simply reject the incoming line (you can thank kafka for this addition ;)

And then ...

well running 20 statsd processes to just handle the load seemed silly, thus ....

Accumulators. which initially "accumulates" lines that can be (thing statsd or carbon-aggrigate) then
emits them to a designated backend, which then can be "PreRegex" to other backends if nessesary
Currently only "graphite" and "statsd" accumulators are available such that one can do statsd -> graphite or even
graphite -> graphite or graphite -> statsd (weird) or statsd -> statsd.  The {same}->{same} are more geared
towards pre-buffering very "loud" inputs so as no to overwhelm the various backends.

NOTE :: if in a cluster of hashers and accumulators .. NTP is your friend .. make sure your clocks are in-sync

And then ...

Data stores, since we have all the data, why not try a more another data store, one that is not so Disk sensitive?
Whisper, Cassandra, Kafka, Mysql, TextFile, {add your own driver}

And then ..

Indexers.  Needs to be able to index the metrics keys in some fashion.  Kafka, Cassandra, LevelDB, "whisper".
(whisper in quotes as it's just a glob on the file system, it need not do anything really, but needs to provide an API
of some kind)

And then ..

HTTP APIs.  Special datastores need to be able to hook into the mighty graphite eco system.  We don't yet want to imitate the
full DSL of graphite, that's really hard.

And then ..

TimeSeries.  The initial pass at external data stores, where "dumb".  Basically things stored a point for every time/value set.
This turned out to very be expensive.  The Whisper format is pretty compact, but requires lots of IO to deal internal rollups and
 seeking to proper places in the RRD.  Cassandra on a "flat point" storage is both very high disk use and for big queries can be
 slow.  Internally things were also not stored effiently causing some ram bloat.  So a variety of compressed time series were
 created to deal w/ all these issues.  These series are binary chunked blobs spanning a time range of points.  Some algorithms are
 very space efficent (gorilla) but not as easy to deal with if slices/sorting, etc are needed.  So given the use case a
 few series types that can be used.

And then ..

Tags (some folks call this "multi-dimensions").  This is the "work in progress point" of where cadent is.


## The beginnings.

Once upon a time ...  there are sometimes (many times) when alarms sound, and as the pager duty recipient, one would have no
idea what it really was (unless it was something easy like this node is down, this process crashed, this NIC is saturated,
this machine is swapping way to much...).  So the team would spend _way_ to much time just tracking what thing had borked
 rather then fixing the issue.  (I think this has happened to others in the 90s early 2ks).

Then came micro-services:  And there were fewer clues as to what was going on.

Metrics:  It's not just for machines anymore.  We decided to make our apps emit all sorts of things about it's
internals.  We said, before your service can go live, there better be a dashboard to indicate what you think are the key
indicators of it's health.

We had to give people tools for that.

Graphite (carbon) + Statsd:  The champions of getting metrics stored and queried quickly.  I hesitate to say almost every beginings
(except those acctually old enough to have started w/ SNMP and RRDtool, and before that tcpdump alone) with these two wonderful tools.

However, it turns out we collect more metrics per second from all the various sources then any sort of actual user trafic (by a few orders
of magnitude).  It, not being customer facing, (not customer facing in the eyes of the business, but certainly "internal" customer
facing) the resources allocated to keeping it running and useful are sparse.

But as much as I love, use, hack, tinker and futz with graphite + statsd ecosphere.  There are three big issues, it's reliance on disks,
single threaded and general performance while trying to keep a few TBs of metrics online and queried, make it difficult to scale (not impossible
by any means), just not easy (lets say i'm happy RRDs have fixed data sizes).

I think is now a common issue in the ecosphere of metrics/ops folks now.  And as a result _many_ projects exist out there in the echo system
that handle lots of "parts" of the problem.  Cadent attempts to merge many of them (it's
"standing on the sholders of giants", uses other good opensource pieces in the wild) in one binary.

Every situation is different, data retention requirement, speed, input volume, query volume, timeording,
persistence solutions, cardinality etc.

Each one comes with it's own cost to run (something alot of projects fail to mention).  For instance if you're just starting,
you're probably not going to have a good sized cassandra/druid cluster and a good sized kafka cluster, as your app
probably runs on one (maybe two) instances (with a couple spares in case of failure of course).

But you probably have a Database somewhere, or at least a hardrive.
As you expand, you start to hit the "wall". There's no way around it.  But let's make moving that wall easier.



# UniqueIDs


## Basic "Series" Name

Internally we maintain this strucutre for a "metric name"

    Tag{
        Name: string
        Value: string
    }

    StatName{
        UniqueID uint64
        UniquIdString string
        Key string
        Tags []Tag
        MetaTags []Tag
        Resolution uint32
        TTL uint32
    }

For all discussions we assume that keys, values are simple ascii w/o punctuation.

## Tagging

Some basic formats, like statsd/graphite don't have tags intrinsict to their formats.  However there are "add on" formats
have been created that allow tagging, these are supported in Cadent

Cadent supports a varity of "tags" formats and will attempt to infer the proper format from the below

    tag=val.tag=val.tag=val
    tag_is_val.tag_is_val
    tag=val,tag=val
    tag_is_val,tag_is_val
    tag=val tag=val
    tag_is_val tag_is_val
    tag:val,tag:val

DONT mix and match those formats (i.e. don't send something like `tag=val tag:val,tag_is_val`).

### Graphite

    <key> <value> <time> name=val name=val ...

### Statsd

This is the so called "datagram" format ..

    <key>:<value>|<type>|@<sample>|#tags:val,tag:val

### Carbon2

    name=val name=val   metaname=val metaname=val <val> <time>

Note the two spaces between the "main tag" set and the metatag set.


## Keys

The internal model is based on the Metrics2.0 spec (http://metrics20.org/spec/) which has a set of `intrinsic tags`
that defined unique-ness in the metrics where as the all the other tags are not included and are simply infered as
metadata, and not part of the unique metric.

The "key" for a metric is defined as follows

### Graphite

    just the <key> from above

### Statsd

    just the <key> from above

### Carbon2

    name_is_val.name_is_val.name_is_val

Where the tags choosen for the key reside in this list

        "host"
        "http_method"
        "http_code"
        "device"
        "unit"
        "what"
        "type"
        "result"
        "stat"
        "bin_max"
        "direction"
        "mtype"
        "unit"
        "file"
        "line"
        "env"
        "dc"
        "zone"

The key is also a SORTED by tag name

Note that we use `_is_` instead of `=` as if porting between various formatting systems (graphite in particular)
the '=' turns out to not go so well in both the internal writing and for URL queries on path names.


## Unique ID

Cadent uses an internal Hash for determining the unique ID which is as follows


    id uint64 = HASH("<key>" + ":" + "<name>=<val> <name>=<val> <name>=<val>")

There are currently 2 hash functions supported

    FNV64a and FarmHash64

The default is `FNV64a` as it's in the go-lang standard libs and it was the default for a while, however, if you're starting
a new install `FarmHash64` is a bit more preformat and may lead to less collisions (that said, nothing has collided yet).

To change the `hash_mode`, in your `prereg.toml` config set `hash_mode="farm"`.

Where again the tags included are *only* the intrinsic ones.

This means that the carbon2 format is actually "doubled up" in a fashion

Also the tag list is also SORTED BY NAME.


### TagMode

The default mode is "metrics2" which means that any tag names that are not in the list above are NOT concidered
for UniqueID ... but this may not work for all systems and use cases, so we have a TagMode of `all` which
basically breaks this intrinsitic tag vs metatags and all tags are concidered `Tags` and there are no `MetaTags`.

An example in the Accumulator/Prereg config file

    [graphite-map]
     listen_server="graphite-in" # which listener to sit in front of  (must be in the main config)
     default_backend="graphite-in"  # failing a match go here

         [graphite-map.accumulator]
         backend = "BLACKHOLE"
         input_format = "carbon2"
         output_format = "carbon2"
         random_ticker_start = false
         tag_mode = "all"  # set to all use all tags for unique ID generation

         accumulate_flush = "1s"
         times = ["1s:1h", "5s:12h", "1m:168h"]



## UniqueID String

Some database systems (cassandra for instance) overflow on a full uint64, so we need to have a format that pretty much
any DB system can understand .. the string/[]char

    id_string string := Base36(id)

Thus we simply convert things to a base 36 "number".

In our DB systems we only store the string version, and use the uint64 version internally (as it's faster).




# TimeSeries


## Basic "Series" Point

A point is really a collection of 6 different numbers plus a time stamp


    StatPoint{
        Time int64
        Min float64
        Max float64
        Sum float64
        Last float64
        Count int64
    }


## Definitions

The core of things for writers/caching (not really used at all in the simply Constist Hashing or Statsd modes).

There are a number of ways for store things.  Some are very good a RAM compression and others are good for ease of use
compatibility, and other internal uses as explained below.

Some definitions:

    - DeltaOfDeltas:

        Since 99.9% of the time "Time" is moving foward there is not need to store the full "int64" each time
        So we store a
        "start" time (T0),
        the next value is then T1 - T0 =  DeltaT1,
        the next value is then T2 - T1 = DeltaT2

        To get a time a the "Nth" point we simply need to

        TN = T0 + Sum(DeltaI, I -> {1, N})

    - VarBit:

        A "Variable Bit encoding" which will store a value in the smallest acctual size it can be stored int

        If the type is an "int64" but the value is only 0 or 1, just store one byte (plus some bits to say what it was)
        if the value is 1000, then only store 2 bytes,
        etc.

        If the type is a float64, but the value is just an int of some kind it will use the int encodings above

    - "Smart"Encoding:

        You'll notice that we store 7 64 bit numbers in the StatMetric.  Sometimes (alot of times) we only have
        a Count==1 in the above which means that all the floats64 are the same (or they better be).  If that's the
        case, we only store one float value (the Sum) (and, depending on the format below, a bit that tells us this fact)

    - TimeResolution:

        Some series (gob, protobuf, gorilla) can use the "int32" for seconds (lowres) of time and
        not int64 (highres) for nanosecond resolution.
        Lowres is the "default" behavior as most of the time our flush times are in the SECONDS not less


## GOB + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The gob format which is a GoLang Specific encoding https://golang.org/pkg/encoding/gob/ that uses the VarBit encoding internally.

This method is pretty efficent.  But it is golang specific so it's not too portable.  But may be good for other uses
(like quick file appenders or something in the future).

## ProtoBuf + VarBit + SmartEncoding + TimeResolution

Standard protobuf encoder github.com/golang/protobuf/proto using the https://github.com/gogo/protobuf generator

Since protobuf is pretty universal at this point (lots of lanuages can read it and use it) it's pretty portable
It's also a bit more efficent in space as the GOB form, due to the nicer encoding methods provided by gogo/protobuf

Also this is NOT time order sensitive, it simply stores each StatMetric "as it is" and it's simple an array
of these internally, so it's good for doing slice operations (i.e. walking time caches and SORTING by times)

This uses "gogofaster" for the generator of proto schemas

    go get github.com/gogo/protobuf/protoc-gen-gogofaster
    protoc --gogofaster_out=. *.proto


## Json, CBOR, BINC

The most portable format, but also the biggest (as we a storing strings in reality).  I'd only use this if you
need extream portability across things, as it's not really space efficent at all.

Cbor and Binc are binary json like formats.  They are more space efficent, but overall, not really worth the encoding
and decoding penatlites.  msgpack/protobufs are a better solution for binary streams.

## MsgPack

In terms of "space" consumed this is a little worse then gogofaster's ProtoBuf.
In terms of Decoding/Encoding speed it's on a level all its own.  Use this for things that favor speed vs space.

## Gorilla + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The most extream compression available.  As well as doing the same goodies mentioned, the core squeezer is the
Float64 compression it uses.  Read up on it here http://www.vldb.org/pvldb/vol8/p1816-teller.pdf.

This does NOT even remotely support out-of-time-order inputs.  The encoding interal to Cadent is a modified version
that allows for multi float encodings, Nano-second encodings and the "smart" encoding as well.

This is by far the best format to store things if your pieces can support it (both in ram and longterm), but due to
the forced timeordering, lack of sorting.  It does not play well w/ many internal things.

The compression is also highly variable depending on incoming values, so it can be hard to "know" what storage or ram
constraints will be needed a-priori (unless you know the domain of your metrics well).

This is probably the best option if you are doing a statsd -> graphite output as time should always be moving
foward.  This one gets tricky with raw graphite inputs as they can be of any time ordering on the incoming.


## Repr

This is the "native" internal format for a metric cadent.  It's NOT very space conciderate (as it's vewry similare
to the json format, but w/ more stuff).  Basically don't use it for storage.


## ZipGob  + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

Instead of a flat []byte buffer for gob encoding use the FLATE buffer (otherwise exactly the same as Gob).  While it
does some some space, due to the internals of the golang Flate, there is alot of GC churn and evils associated
with this one if used for a large number of series.  So this may be a good "final persist state" but it should get converted
out of this format for use elsewhere.

### *DON'T USE THIS*

Why is it included?  It may be handy for other things, but in terms of handling massive volumes, this is not
very good due to the MASSIVE GC/byte pressure it exerts.



# The PointType Enum

For reference as we store a uint8 in data stores.  I'd only recommend using Gob, Protobuf or Gorilla, unless you
need "super compatability" then JSON/MSGPACK may be a good bet.  Whatever you do don't really use the ZipGob.

    const (
        GOB uint8 = iota + 1   # 1
        ZIPGOB   # 2
        JSON # 3
        PROTOBUF  # 4
        GORILLA # 5
        MSGPACK # 6
        BINC # 7
        CBOR # 8
        REPR # 9
    )


# Binary Format

In Cadent each series is stored in binary blobs, and each format has a different format.  For Gob, Msgpack, protobuf,
 cbor, json, and binc, these formats are taken care of by their associated libraries out in the wild.

However each binary blob incudes a 4 byte header to indicate what type of blob it really is.

Protobuf, Repr (which is encoded as json), and JSON do NOT have these headers, as they are "self describing". Since internllay
they handle the difference in low/high res and "smart" encodings with out any other manipulation.

So for all the formats each begins

    gobn -> gob high res format (times are unix nanosecond timestamps)
    gobl -> gob low res format (times are unix time stamps)
    gors -> gorilla low res
    gort -> gorilla high res
    mspl -> msgpack low res
    msph -> msgpack high res
    zbts -> zipgob high res
    zbtl -> zipgob low res
    cdbh -> binc high res
    cdbl -> binc low res
    cdch -> cbor high res
    cdcl -> cbor low res

## Gorilla

The most compilcated (and also the best compression both in RAM and on disk) is the gorialla format.
I'll attempt to describe it's internals a little bit.

Adapted and manipulated from the original algo on https://github.com/dgryski/go-tsz
but modified to handle "multiple" values the original is simply

    T,V

but we need T,V,V,V,V ...

We could do a multi array of T,V, T,V ... but why store time that many times?

https://github.com/dgryski/go-tsz also uses uint32 for the base time.

Since the Nano-second precision is nice, but "very" variable.  Meaning the delta-of-deltas
can be very large for given point (basically base-time of (int32) appended to sub-second(int32)
we break up the time "compression" in to 2 chunks .. the first "epoch" time and then the subsecond part.

The highly variable part is the sub-second one, and almost always going to be not-so-compressible

Basically we need to take an int like 1467946279766433748 and split it into 2 values

1467946279 and 766433748

If our resolution dump is in the "second" range (i.e. 99% of cases)
Most of the time the deltas of the 2nd half will be "0" so we only really need to store one bit "0"

We use the golang time module to do the splitting and re-combo, as well, it's good at it.

Since the "nanosecond" part of the time stamps is highly fluctuating, and not going to be "in order"
(the deltas will be negative many times)
we treat that part like a "value" as apposed to a "timestamp" and compress it like a normal float64

Format ..

First the header

    [4 byte header = gors|gort][1 byte number of values| 1 or 5]

The next bit is the starting time

    [32 bit T0][32 bit Tms] for "high res"

    [32 bit T0] for "low res"

Then the "initial start" which is the full int64/float64 we use as the starting values
(the time value is a delta of the first T0)

    [DelTs][XorTms0][v0][v1][...]

The `Xor` in the above signifies the floating point compression.  It's a variable bit
encoding and more info can be found here http://www.vldb.org/pvldb/vol8/p1816-teller.pdf

Next for high res formats

    [NumBit][DoDTs][XorTms0][XorV0][XorV1][...]

And for low res formats

    [NumBit][DoDTs][XorV0][XorV1][...]

And a third Finally for the "smart" encoding we need an extra bit to determine if there are 1 or 5 values

    [NumBit][DoDTs][small|fullbit][XorV0][XorV1][...]

The "end" of the stream is

    [0x04][0xffffff][0]

Below is the default format used in Cadent put all together

    [4 byte header = gors][1 byte number of values|5][32 bit T0]
    [DelTs][XorTms0][min|float64][max|float64][last|float64][sum|float64][count|int64]
    [NumBit][DoDTs][fullbit|1][min|Xor][max|Xor][last|Xor][sum|Xor][count|Xor]  # for the full
    ...
    [NumBit][DoDTs][smallbit|0][sum|Xor]  # for the smart

    ...
    [0x04][0xffffff][0]






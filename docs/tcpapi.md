
# TCP API (alpha)

note: in alpha mode, it has not been throughly "stress tested".

Like the [http api](./api.md), there is a simply TCP protocal for finding and getting metrics
without all that overhead of HTTP "stuff", easier for pooling and general performance.

A few basic commands :::    

Just `echo HELP | nc host port`


    Cadent TCP reader interface

    HELP -- this message
    PING -- PONG
    GETM <path> [<from time?> <to time?> <resolution?> <format?>] -- get raw metric data
    FINDM <path> [<format?>] -- find metric names
    GETC <path> [<from time?> <to time?> <format?>] -- get metric data from cache (if supported)
    GETCB <path> -- get metric binary timeseries from cache (if supported)
    INCA <path> -- Is this metric in the cache? (returns 0,1)
    CLIST [<format?>] -- List of stat names ([]repr.StatName) in the cache currently
    EXIT -- terminate connection

     *NOTE: GETCB can only take an exact "match" of

    [...] == optional in order, but the order maters
    <format> == json (default), msgpack, protobuf
    <path> == something like "stats.host.cpu.total"

    <from>, <to> == A parseable date string or time stamp or relative
        i.e :: 1480731100, -1h, now, 2015-07-01T20:10:30.781Z

      default <from> == -1h
      default <to> == now
      default <resolution> == 1 (or the minimum set in the config)
      default <format> == json

    - parameters are space separated

    - End of any stream/output is indicated by "\r\n"

    - Errors are : '-ERR {code} {message} \r\n'

    - BINARY protocals (GETCB, protobuf, and msgpack) have a length prefix
      So when reading in make sure to add 2 extra bytes for the terminator
      [uint32|bigendian][...content...]\r\n


    



NOTE: tags are not yet supported for the TCP api


## Typical config

Looks alot like the httpapi config

    [graphite-proxy-map.accumulator.tcpapi]
        listen = "0.0.0.0:8084"
        max_clients=8192  # max clients allowed
        num_workers=16  # max things that can be running at any given time
        # if wanting a TLS tcp cserver
        # key = /path/to/key
        # cert = /path/to/cert
        
        
        # you can references the metrics/indexer sections OR you can define an entire other 
        use_metrics="my-metrics-writer-name"
        use_indexer="my-indexer-writer-name"
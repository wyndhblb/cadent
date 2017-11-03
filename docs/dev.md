
Testing and Dev
---------------

### pprof data + stats

### config section

    [profile]
    ## Turn on CPU/Mem profiling
    enabled=true

    ##  there will be a http server set up to yank pprof data on :6060

    listen="0.0.0.0:6060"
    rate=100000
    block_profile=false # this is very expensive


### handy urls

    ram: http://localhost:6060/debug/pprof/heap
    profile: http://localhost:6060/debug/pprof/profile
    stats: http://localhost:6060/debug/vars  -- has all internal counters and a "snapshot" current statds data


Some quick refs for performance and other "leaky/ram" usages for tuning your brains
(we've the profiler stuff hooked)

    
    go tool pprof  --inuse_space --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --inuse_objects --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --alloc_space --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --alloc_objects --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/profile
    
    
    # just how many mutex locks are stuck?
    curl http://127.0.0.1:6061/debug/pprof/goroutine?debug=2
    

## EchoServer

things come with an "echo server" which is simply what it is .. just echos what it gets to stdout

the make will make that as well, to run and listen on 3 UDP ports

    echoserver --servers=udp://127.0.0.1:6002,udp://127.0.0.1:6003,udp://127.0.0.1:6004
    
    # 3 tcp servers
    echoserver --servers=tcp://127.0.0.1:6002,tcp://127.0.0.1:6003,tcp://127.0.0.1:6004
    

## StatBlast

There is also a "line msg" generator "statblast." It will make a bunch of random stats based on the `-words` given

To simulate a "real" metric input, things will be prefixed randomly with 

    stats.
    stats.gauges.
    stats.timers.
    stats.set.
    stat_counts.
    server.
    
And suffixed with

    {random word}
    count
    min
    max
    upper
    mean
    ...etc
    

Usage ... 
   
    Usage of statblast:
          -buffer int
                send buffer (default 512)
          -forks int
                number of concurrent senders (default 2)
          -justone string
                just fire the same 'key' (defined here) not a random one (values will be random)
          -notick
                don't print the stats of the number of lines sent (if servers==stdout this is true)
          -rate string
                fire rate for stat lines (default "0.1s")
          -servers string
                list of servers to open
                (stdout,tcp://127.0.0.1:6002,tcp://127.0.0.1:6003), you can choose
                tcp://, udp://, http://, unix://, kafka://
        
                kafka type should be a list of brokers (kafka://192.168.0.2:9092/topic/{json,msgpack,protobuf})
                 (default "tcp://127.0.0.1:8125")
          -type string
                statsd or graphite or carbon2 or json (default "statsd")
          -words string
                compose the stat keys from these words (default "test,house,here,there,badline,cow,now")
          -words_per_stat int
                make stat keys this long (moo.goo.loo) (default 3)


## ReadBlast

Like statblast but for reading from either the TCP or HTTP api enpoints

    Usage of readblast:
        -forks int
            number of concurrent senders (default 2)
        -format string
            json, msgpack, protobuf (default "json")
        -from int
            number of hours back to get data from (default -1)
        -metric string
            list of metrics to read 'my.metric,this.metric,that.metric' (required)
        -server string
            server to blast (tcp://127.0.0.1:8084, http://127.0.0.1:8084) (default "http://127.0.0.1:8083/graphite/rawrender")
        -tick
            print stats (default true)

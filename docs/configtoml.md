
# config.toml


This is the generic configuration for specifying general configuration and the various front end endpoints for accepting metrics formats, 
consistent hashing and relay endpoints.

   
    
    #############################
    ## Logging
    #############################
    [log]
    
    # LogFormat: log line format
    format = """%{time:2006-01-02 15:04:05.000Z07:00} %{level:.4s} %{id} [%{module}] (%{shortfile}) - %{message}"""
    # of if you want some structured bits
    # format ="""{time="%{time:2006-01-02T15:04:05.000Z07:00}", module="%{module}", file="%{shortfile}", id="%{id}", level="%{level:.6s}", message="%{message}"}"""
    
    # LogLevel: oggin level (DEBUG|INFO|NOTICE|WARN|ERROR|CRITICAL)
    level = "$ENV{LOG_LEVEL:INFO}"
    
    # LogFile: where to log things (stdout|stderr|{path/to/file})
    file="$ENV{LOG_FILE:stdout}"
    
    
    #############################
    ## System -- some golang settings
    #############################
    [system]
    
    # PidFile: PID file to check if things is running already
    pid_file="$ENV{PID_FILE:/opt/cadent/cadent.pid}"
    
    # Cores: performance tuning suggest that n - 2 or n -1 is best to avoid over creation
    # default will be to use all cores
    num_procs=$ENV{GOPROCS:}
    
    # Garbage Collection Persent: if heap grows by this amount (%) trigger a GC
    # same as GOGC env param, I have found for 60% to be a good number for high volume inputs
    gc_percent=$ENV{GOGC:60}
    
    # Release Ram Time: force a release of free ram ever "X" min
    free_mem_tick=$ENV{GOGC_FORCE_FREE_MEM:5}
    
    
    #############################
    ## Profiling -- pprof
    #############################
    
    [profile]
    
    # Profile Enabled:  Turn on CPU/Mem profiling
    enabled=true
    
    # Profile Http Endpoint: there will be a http server set up to yank pprof data on :6060
    listen="0.0.0.0:6060"
    
    # Profile Sampleing rate:
    rate=100000
    
    # Profile Block Profile: this is _very_ expensive use only for debugging, or general performance tracing
    block_profile=false 
    
    #############################
    ## Internal Stats -- expose some internal stats (and a little status page)
    ##  note: this page is only for the incoming metrics, not anything in the pregreg, statsd is used for those
    #############################
    
    [health]
    
    # Enable: turn this on
    enabled=true
    
    # Internal Stats Endpoint: there will be an internal health http server set up as well
    # will respond to '/ops/status' -> "GET OK"
    # will respond to '/ops/ping' -> "GET OK"
    # '/stats' --> blob of json data (adding ?jsonp=xxx will return jsonp data)
    # '/' --> a little file that charts (poorly) the above stats
    listen="0.0.0.0:6061"
    
    # Stored Points: stats are "rendered" every 5 seconds, so this will keep last X points
    points=500
    
    # TLS: https this server if desired
    # key="/tmp/server.key"
    # cert="/tmp/server.crt"
    
    
    #############################
    ## Emit To Internal goodies to Statsd 
    ##  all statsd get the hostname pre-pended to outgoing metrics
    ##  {host}.{prefix}.{my.stat}
    #############################
    
    [statsd]
    # Statsd Host: fire out some internal to statsd if desired
    server="127.0.0.1:8125"
    
    # Statsd Prefix
    prefix="cadent"
    
    # Statsd flush internval: the internal client is buffered and will send metrics either 
    # when the buffer hits 512 bytes or this timer is reached
    interval=1  # send to statd every second (buffered)
    
    # Statsd Sample Rates
    # There are 2 clients a "slow" and a "fast" internally
    # 'slow' is one is used for functions/counters that are relatively quite (no sampling rate) 
    # 'fast' is used for functions that respond in the microsecond range and/or are called "alot"
    # if you do not put a sample rate below, 70% of your CPU will be eaten by just statsd things (not good)
    
    # Statsd Fast Timer Sample Rates
    timer_sample_rate=0.01
    # Statsd Fast Counter Sample Rates
    sample_rate=0.1
    
    # DO NOT include the host in the outgoing
    # 
    # by defaults we prefix all metrics with {prefix}.{hostname-long}.
    # if you do not wish to have the host name set this to true
    # do_not_include_host = true 
    
    # HostName size
    # 
    # if including hostname, either choose the "short" hostname or "long"
    # short: myhost
    # long: myhost.base.com
    # use_short_hostname = true 
    
    
    #############################
    ## Gossiper -- (used for clustered API)
    #############################
    [gossip]
    
    # Gossip enabled:
    enabled=true
    
    # Gossip mode: (local|lan|wan)
    mode="lan"
    
    # Gossip port number:
    port=8889
    
    # Gossip Node Name: will pick the hostname as default and should be unique, i recommend putting something
    # name = ""
    
    # Gossip Seed: a node that this node will initially hook up with to find the memebers
    # seed = ""
    
    # Gossip Bind: default is 0.0.0.0
    # bind = ""
    
    # Addr to advertise as ourselves (default to first interface which is usually the loopback .. so set this)
    # advertise_address = ""
    
    
    #############################
    ## SERVERS -- Incoming protocals
    #############################
    [servers]
    
    #############################
    ## Incoming server defaults
    ##  All these defaults apply to any subsections, can be overridden in each section
    #############################
    
    
    [servers.default]
    
    # Statsd Tick: if true, will emit the serviers state of stats every 5 seconds
    # can add alot of noise to the logs but good for general info
    stats_tick=false
    
    # LRU Cache: 
    # cache keys/server pairs in the consistent hashing parts in an LRU mem cache for speed
    cache_items=500000
    
    ###
    ### Health Checking options
    ###
    
    # Heartbeat Tick: check relay server every X seconds
    heartbeat_time_delay=60
    
    # Heartbeat Timeout: Timeout for failed connections (in Seconds)
    heartbeat_time_timeout=1
    
    # Heartbeat Fails: if a server has failed "X" times it's out of the loop
    failed_heartbeat_count=3
    
    # Server Down policy: If we detect a downed node,
    # `remove_node` -- we can REMOVE the node from the relay pool, which means stats will get redirected to a new host
    # `ignore` -- stop checking the node, but don't remove from the relay pool !!this will start to DROP stats that would have gone to that hose!!
    server_down_policy="ignore"
    
    ###
    ### Relay Output options
    ###
    
    # Send Method:
    # `bufferedpool` (default) : uses `max_pool_connections` and buffers `pool_buffersize` bytes before sending to a relay server
    # `pool`: uses `max_pool_connections` and sends each hashed item as they come in (no buffering)
    # 
    sending_method = "bufferedpool"
    
    # Outgoing Pool Size: outgoing connections are pooled per outgoing server
    max_pool_connections=10
    
    # Pool Buffer Size (512 default): used only if sending_method==bufferedpool
    # recommendations
    # - Lossy/WAN UDP : 512
    # - JumpFrames/LAN UDP: 2048
    # - TCP: 8096 (or more)
    pool_buffersize=1024
    
    ###
    ### Incoming Buffers
    ###
    
    # Incoming Read Buffer: size in bytes of any socket buffer
    # for UDP inputs this should be a large number as there is "one" socket (1MB is good)
    # for TCP a much smaller one is good as there can be MANY tcp sockets
    read_buffer_size=1048576  # UDP
    # read_buffer_size=8192  # TCP
    
    # Max Read Buffer in Bytes: as we injest data, we fill up an internal buffer, if that buffer 
    # fills up, we are not keeping up, so we need to backpressure or drop tings
    # (for UDP, there is no concept of backpressure)
    max_read_buffer_size=819200
    
    ###
    ### Workers
    ###  A dispatch queue is used to handle all the incoming data 
    ###  the more workers you have the more you can process, at the expense of CPU
    
    # Incoming Workers: Do NOT make this too high, or you'll end up is Lock Contention   
    workers=10
    
    # Output Workers: Number of workers to handle the outputs to relays
    # since this is a network action (lots of time spent waiting), more workers here are ok
    out_workers=32
    
    
    ################
    ## Example Statsd Consistent/Proxy server
    ################
    
    [servers.statsd-proxy]
    
    # Listen: what socket port are we listening to
    listen="udp://0.0.0.0:8125"
    
    # Message Type: what kind of message/line protocal is the incoming
    # can be: statsd, graphite, carbon2, opentsdb, json
    msg_type="statsd"
    
    # Relay Servers: The servers in your hash ring
    [[statsd-proxy.servers]]
    
    # Server List: {sockettype}://{host}:{port}
    servers=["udp://host1:8126", "udp://host2:8126", "udp://host3:8126"]
    
    # HealthCheck Endpoints:
    #  since UDP is not pingable, you'll need to specifiy the things to ping
    #  in this example, we use the internal stats port (set above) if relaying to other cadent(s)
    #  NOTE: there need to be as many of these as servers above
    check_servers = ["tcp://host1:6061", "tcp://host2:6061", "tcp://host3:6061"]
    
    ###
    ### REPLICATION: you can have as many `[[statsd-proxy.servers]]` sections, each one will REPLICATE the incoming lines
    ###  to a different hash ring
    # [[statsd-proxy.servers]]
        
    # servers=["udp://otherhost1:8126", "udp://otherhost2:8126"]
    # check_servers = ["tcp://otherhost1:6061", "tcp://otherhost2:6061"]
    
    ################
    ## Example Statsd Server (prereg.toml is needed for this)
    ################
    
    # this example uses the "prereg" configs to farm the incoming to a statsd accumulator
    # that accumulator then farms things to a graphite backend (defined below)
    [servers.statsd]
    
    # Listen: what socket port are we listening to
    listen="udp://0.0.0.0:8126"
    
    # Message Type: what kind of message/line protocal is the incoming
    msg_type="statsd"
    
    # OutDevNull: since everything is being relayed to the graphite endpoint, we don't really do any hashing
    out_dev_null=true
    
    
    ################
    ## Example Graphite - Consistent/Proxy server
    ################
    [servers.graphite-proxy]
    
    # Listen: what socket port are we listening to
    listen="tcp://0.0.0.0:2003"
    
    # Message Type: what kind of message/line protocal is the incoming
    # can be: statsd, graphite, carbon2, opentsdb, json
    msg_type="graphite"

    # override default number of input workers
    workers=5
    
    # override default outgoing buffer size (since this is TCP we can have a bigger one)
    pool_buffersize=16384
    
    # override TCP buffers should be smaller then UDP as there can be Many TCP connnections
    read_buffer_size=8192
    max_read_buffer_size=819200
    
    # Relay Servers: The servers in your hash ring
    [[graphite-proxy.servers]]
    
    # Server List: {sockettype}://{host}:{port}
    servers=["tcp://host1:2006", "tcp://host2:2006", "tcp://host3:2006"]
    
    # HealthCheck Endpoints: since the above is TCP, we don't need to specify any here
    # check_servers = ["tcp://host1:2006", "tcp://host2:2006", "tcp://host3:2006"]
    
    ### REPLICATION: 
    # [[graphite-proxy.servers]]
    # servers=["tcp://otherhost1:2006", "udp://otherhost2:2006"]
    
    ################
    ## Example Graphite Server (prereg.toml is needed for this)
    ################
    
    # this example uses the "prereg" configs to farm the incoming to a statsd accumulator
    # that accumulator then farms things to a graphite backend (defined below)
    [servers.graphite]
    
    # Listen: what socket port are we listening to
    listen="tcp://0.0.0.0:2006"
    
    # Message Type: what kind of message/line protocal is the incoming
    msg_type="graphite"
    
    # override default number of input workers
    workers=5
    
    # override TCP buffers should be smaller then UDP as there can be Many TCP connnections
    read_buffer_size=8192
    max_read_buffer_size=819200
    
    # OutDevNull: since everything is being relayed to the graphite endpoint, we don't really do any hashing
    out_dev_null=true
    
    

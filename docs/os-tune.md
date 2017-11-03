

OS tuning
=========

## Consistent Hashing relay

Consistent hashing is very CPU and IO intensive (computing many 100s of thousands of hashes and forwarding them per second). 
The below are some good practices for getting all the muster you can from your nodes.

If you expect ~500-1million/lines per second that need to be consistently hashed coming from UDP sockets .. you will have to tune your
network stack .. 

Here is what I recommend.  Lots of buffers basically and larger queues.

    sysctl -w net.core.rmem_max=67108864
    sysctl -w net.core.wmem_max=67108864
    sysctl -w net.core.rmem_default=67108864
    sysctl -w net.core.wmem_default=8388608
    sysctl -w net.ipv4.tcp_rmem='4096 87380 8388608'
    sysctl -w net.ipv4.tcp_wmem='4096 65536 8388608'
    sysctl -w net.ipv4.tcp_mem='8388608 8388608 8388608'
    sysctl -w net.ipv4.udp_mem='8388608 8388608 8388608'
    sysctl -w net.ipv4.udp_rmem_min=67108864
    sysctl -w net.ipv4.tcp_congestion_control=htcp

    sysctl -w net.core.netdev_max_backlog=10240
    sysctl -w net.ipv4.tcp_max_syn_backlog=10240
    sysctl -w net.ipv4.tcp_rfc1337=1
    sysctl -w net.core.somaxconn=4096
    sysctl -w net.ipv4.tcp_congestion_control=htcp
    sysctl -w net.ipv4.route.flush=1

Another good process is to give things more CPU/IO priority

    renice 5 -p $(pgrep cadent)
       
In your config.toml file for the consistent hashing i would also use 

    read_buffer_size=67108864  # this is the max value from net.core.rmem_default above
    
You can expect the server to then consume about `read_buffer_size * workers` in ram, that is expected as the large buffers are
pre allocated.
    
Using the SO_REUSEADDR socket abilities lets you have multiple binds to the same port, and since most of the time they are
idle doing IO work, you can expand these to more listeners to spread the goodies around.

    workers={2 or 4 * NumCores}
    out_workers={2 or 4 * NumCores}

It's a good idea to have the receivers of the consistently hashed items go to TCP endpoints (not UDP)

    servers=["tcp://172.21.5.89:8128","tcp://172.21.4.128:8128","tcp://172.21.6.137:8128"]

And increase the buffer sending size, as it's TCP which will save lots of cycles on both the receiver statsd and sender

    pool_buffersize=16384

I have also found that, since there can be too few threads spawned using the default Go Lang CPUs in use equal to the 
number of cores on the system.  Since much of the work here is all network, and the threads are sitting idle(ish) alot of the 
time, a way to tell Go to boost the thread count is to force a high CPU count.  I typically double the real cpu count

    num_cpu={2 * NumCores}

These settings are for a "stand alone" consistent hashing server.  

In production we run a 3 node hashing relay with the statsd client (also cadent, just a different process) on each node, 
so the relay farm basically hashes to themselves on a different port, and is able to deal effectively with ~500-750k UDP packets/second
with very minimal UDP drops (they do happen, mostly, even with the large buffers, incoming services can huge blasts that overwhelm the buffers.
Such is the penalty of udp.)

# Consistent

A little lib for dealing with consistent hashing rings and the various hashing functions that can be used

## use

```
import "github.com/wyndhblb/consistent"

func main(){
    hashRing := consistent.New()
    hashRing.SetHasherByName("mmh3") // set the hashing function
    hashRing.SetNumberOfReplicas(100)  // set the number of replicas (the hash ring pie slices)
    hashRing.SetElterByName("statsd")  // "elt" (when adding determining the server, run a format like funtion)
    
    // add some servers
    // !NOTE THE ORDERING MATTERS!
    hashRing.Add("my1.server.is.great.com")
    hashRing.Add("my2.server.is.great.com")
    hashRing.Add("my3.server.is.great.com")
    
    // remove some servers
    hashRing.Remove("my3.server.is.great.com")
    
    // get a server for this string key
    toServer := hashRing.Get("my-nice-thing")
    
    // get N items closest to the key
    // good for replication purposes
    toServers := hashRing.GetN("my-nice-thing", 2)
}
```

## Hashers

Available hashing functions are

- crc, crc32: CRC32 (default)
- md5, graphite: MD5
- sha1: SHA1
- fnv, fnv32: FNV32
- fnva, fnv32a: FNVa32
- statsd, nodejs-hashring: a "node.js" imitation of nodejs-hashring (used in statsd)
- mmh, mmh3, mmh32: MurMurHash 32 bit

## Elters

Available Elters

All servers will be cleaned like so to keep the same hashing should the protocal change

```
(tcp|udp|http)://{ipadd}:{port} -> {ip}:{port}
```

- statsd:  IPs/Names into: `{server}:{index}`
- graphite:  IPs/Names into: `{server}-{index}`
- default:  IPs/Names into: `{server}{index}`

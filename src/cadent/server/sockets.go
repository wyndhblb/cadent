/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
   To allow net.UDPcon to have various Nice kernel flags used

*/

package cadent

import (
	"github.com/wyndhblb/go_reuseport"
	"net"
)

// GetReuseListener for TCP conns
func GetReuseListener(protocol string, addr string) (net.Listener, error) {
	return reuseport.Listen(protocol, addr)
}

// GetReuseListenerWithBuffer for TCP conns
func GetReuseListenerWithBuffer(protocol string, addr string, readBuffer int, writeBuffer int) (net.Listener, error) {
	return reuseport.ListenWithBuffer(protocol, addr, readBuffer, writeBuffer)

}

// GetReusePacketListener for UDP conns
func GetReusePacketListener(protocol string, addr string) (net.PacketConn, error) {
	return reuseport.ListenPacket(protocol, addr)
}

// GetReusePacketListenerWithBuffer for UDP conns
func GetReusePacketListenerWithBuffer(protocol string, addr string, readBuffer int, writeBuffer int) (net.PacketConn, error) {
	return reuseport.ListenPacketWithBuffer(protocol, addr, readBuffer, writeBuffer)
}

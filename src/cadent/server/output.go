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
   The various Output Pool writers objects/functions
*/

package cadent

import (
	"cadent/server/dispatch"
	"cadent/server/netpool"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"fmt"
	"net"
	"net/url"
	"time"
)

// OutMessageWriter interface for output writers
type OutMessageWriter interface {
	Write(*OutputMessage) error
}

// PoolWriter a writer that does uses a pool of outgoing sockets for sending
type PoolWriter struct{}

// Write send the message to the backend pool sockets
func (p *PoolWriter) Write(j *OutputMessage) error {
	// to send stat lines via a pool of connections
	// rather then one socket per stat
	defer stats.StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	var outsrv netpool.NetpoolInterface
	var ok bool

	// lock out Outpool map
	if j == nil {
		return nil
	}

	make_pool := func() int {
		j.server.poolmu.Lock()
		defer j.server.poolmu.Unlock()

		if outsrv, ok = j.server.Outpool[j.outserver]; ok {
			ok = true
			return 0
		} else {
			mURL, err := url.Parse(j.outserver)
			if err != nil {
				stats.StatsdClient.Incr("failed.bad-url", 1)
				j.server.FailSendCount.Up(1)
				j.server.log.Error("Error sending to backend Invalid URL `%s` %s", j.outserver, err)
				return 2 //cannot retry this
			}
			if len(j.server.Outpool) == 0 {
				j.server.Outpool = make(map[string]netpool.NetpoolInterface)

			}
			if j.server.SendingConnectionMethod == "bufferedpool" {
				outsrv = netpool.NewBufferedNetpool(mURL.Scheme, mURL.Host+mURL.Path, j.server.WriteBufferPoolSize)
			} else {
				outsrv = netpool.NewNetpool(mURL.Scheme, mURL.Host+mURL.Path)
			}
			if j.server.NetPoolConnections > 0 {
				outsrv.SetMaxConnections(j.server.NetPoolConnections)
			}
			// populate it
			err = outsrv.InitPool()
			if err != nil {
				j.server.log.Warning("Poll init error %s", err)
				return 1
			}
			j.server.Outpool[j.outserver] = outsrv
			return 0
		}
	}

	// keep retrying
	retcode := make_pool()
	if retcode == 1 {
		time.Sleep(time.Second)
		return fmt.Errorf("Pool %s failed to intialize, check the outgoing servers for 'aliveness'", j.outserver)
	} else if retcode == 2 {
		return fmt.Errorf("Pool %s failed to intialize, Hard failure", j.outserver)
	}

	netconn, err := outsrv.Open()
	defer outsrv.Close(netconn)

	if err != nil {
		stats.StatsdClient.Incr("failed.bad-connection", 1)
		j.server.FailSendCount.Up(1)
		return fmt.Errorf("Error sending to backend %s", err)
	}
	if netconn.Conn() != nil {
		// Conn.Write will raise a timeout error after 1 seconds
		netconn.SetWriteDeadline(time.Now().Add(j.server.WriteTimeout))
		//var wrote int
		toSend := append(j.param, repr.NEWLINE_SEPARATOR_BYTE)
		by, err := netconn.Write(toSend)

		//log.Printf("SEND %s %s", wrote, err)
		if err != nil {
			stats.StatsdClient.Incr("failed.connection-timeout", 1)
			j.server.FailSendCount.Up(1)
			outsrv.ResetConn(netconn)
			return fmt.Errorf("Error sending (writing) to backend: %s", err)
		} else {
			j.server.BytesWrittenCount.Up(uint64(by))
			j.server.SuccessSendCount.Up(1)
			stats.StatsdClient.Incr("success.send", 1)
			stats.StatsdClient.Incr("success.sent-bytes", int64(len(toSend)))
		}

	} else {
		stats.StatsdClient.Incr("failed.aborted-connection", 1)
		j.server.FailSendCount.Up(1)
		return fmt.Errorf("Error sending (writing connection gone) to backend: %s", j.outserver)
	}
	return nil
}

/** SingleWriter a writer that does not use a buffered pool for sending to sockets **/
type SingleWriter struct{}

func (p *SingleWriter) Write(j *OutputMessage) error {
	//this is for using a simple tcp connection per stat we send out
	//one can quickly run out of sockets if this is used under high load
	if j == nil {
		return nil
	}

	defer stats.StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	m_url, err := url.Parse(j.outserver)
	if err != nil {
		stats.StatsdClient.Incr("failed.bad-url", 1)
		j.server.FailSendCount.Up(1)
		return fmt.Errorf("Error sending to backend Invalid URL %s", err)
	}
	conn, err := net.DialTimeout(m_url.Scheme, m_url.Host+m_url.Path, 5*time.Second)
	if conn != nil {

		conn.SetWriteDeadline(time.Now().Add(j.server.WriteTimeout))
		//send it and close it
		to_send := append(j.param, repr.NEWLINE_SEPARATOR_BYTE)
		_, err = conn.Write(to_send)
		conn.Close()
		conn = nil
		if err != nil {
			stats.StatsdClient.Incr("failed.bad-connection", 1)
			j.server.FailSendCount.Up(1)
			return fmt.Errorf("Error sending (writing) to backend: %s", err)
		}
		stats.StatsdClient.Incr("success.sent", 1)
		stats.StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
		j.server.SuccessSendCount.Up(1)
	} else {
		j.server.FailSendCount.Up(1)
		return fmt.Errorf("Error sending (connection) to backend: %s", err)
	}
	return nil
}

/***** OutputDispatchJob Dispatcher Job work queue ****/
type OutputDispatchJob struct {
	Writer  OutMessageWriter
	Message *OutputMessage
	Retry   int
}

func (o *OutputDispatchJob) IncRetry() int {
	o.Retry++
	return o.Retry
}

func (o *OutputDispatchJob) OnRetry() int {
	return o.Retry
}

func (o *OutputDispatchJob) DoWork() error {
	err := o.Writer.Write(o.Message)

	if err != nil {
		o.Message.server.log.Error("%s", err)
		if o.OnRetry() < 2 {
			o.Message.server.log.Warning("Retrying message: %s retry #%d", o.Message.param, o.OnRetry())
		}
	}
	return err
}

func NewOutputDispatcher(workers int) *dispatch.Dispatch {
	writeQueue := make(chan dispatch.IJob, workers*10) // a little buffer
	dispatchQueue := make(chan chan dispatch.IJob, workers)
	writeDispatcher := dispatch.NewDispatch(workers, dispatchQueue, writeQueue)
	writeDispatcher.SetRetries(2)
	return writeDispatcher
}

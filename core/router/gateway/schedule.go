// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package gateway

import (
	"sync"
	"time"

	ttnlog "github.com/TheThingsNetwork/go-utils/log"
	"github.com/TheThingsNetwork/go-utils/queue"
	"github.com/TheThingsNetwork/ttn/api/fields"
	pb_lorawan "github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	router_pb "github.com/TheThingsNetwork/ttn/api/router"
	"github.com/TheThingsNetwork/ttn/utils/errors"
	"github.com/TheThingsNetwork/ttn/utils/random"
	"github.com/TheThingsNetwork/ttn/utils/toa"
)

type scheduledItem struct {
	mu sync.RWMutex

	id string

	time      time.Time
	timestamp uint64 // microseconds - full
	duration  uint32 // microseconds

	payload *router_pb.DownlinkMessage
}

func (i *scheduledItem) Time() time.Time {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.time
}
func (i *scheduledItem) Timestamp() int64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return int64(i.timestamp) * int64(time.Microsecond)
}
func (i *scheduledItem) Duration() time.Duration {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return time.Duration(i.duration) * time.Microsecond
}

const maxUint32 = 1 << 32

func getFullTimestamp(full uint64, lsb uint32) uint64 {
	if int(lsb)-int(full) > 0 {
		return uint64(lsb)
	}
	if uint32(full%maxUint32) <= lsb {
		return uint64(lsb) + (full/maxUint32)*maxUint32
	}
	return uint64(lsb) + ((full/maxUint32)+1)*maxUint32
}

var ErrScheduleInactive = errors.New("gateway: schedule not active")

type Schedule struct {
	sync.RWMutex

	gateway *Gateway

	ctx ttnlog.Interface

	timestamp uint64
	offset    time.Duration

	items    map[string]*scheduledItem
	schedule queue.Schedule
	queue    queue.JIT

	downlink    chan *router_pb.DownlinkMessage
	subscribers int
}

// NewSchedule creates a new Schedule
func NewSchedule(ctx ttnlog.Interface) *Schedule {
	s := &Schedule{
		ctx: ctx,
	}

	return s
}

func (s *Schedule) init() {
	s.schedule = queue.NewSchedule()
	s.queue = queue.NewJIT()
	s.items = make(map[string]*scheduledItem)
	s.downlink = make(chan *router_pb.DownlinkMessage)

	// Send downlink to downlink channel
	go func() {
		for {
			nextI := s.queue.Next()
			if nextI == nil {
				break
			}
			next := nextI.(*scheduledItem)
			select {
			case s.downlink <- next.payload:
			default:
			}
		}
		close(s.downlink)
		s.downlink = nil
	}()

	// Clean up expired items
	go func() {
		for {
			expiredI := s.schedule.Next()
			if expiredI == nil {
				break
			}
		}
		s.Lock()
		s.queue.Destroy()
		s.items = nil
		s.queue = nil
		s.schedule = nil
		s.Unlock()
	}()
}

// Sync the gateway schedule
func (s *Schedule) Sync(timestamp uint32) {
	s.Lock()
	defer s.Unlock()
	if s.timestamp == 0 {
		s.timestamp = uint64(timestamp)
	} else {
		s.timestamp = getFullTimestamp(s.timestamp, timestamp)
	}
	s.offset = time.Duration(time.Now().UnixNano() - int64(s.timestamp*1000))
}

func (s *Schedule) getFullTimestamp(lsb uint32) uint64 {
	return getFullTimestamp(s.timestamp, lsb)
}

func (s *Schedule) getRealtime(fullTimestamp uint64) time.Time {
	return time.Unix(0, int64(s.offset)+int64(fullTimestamp)*1000)
}

var DefaultGatewayRTT = 100 * time.Millisecond
var DefaultGatewayBufferTime = 500 * time.Millisecond // TODO: add this to gateway status message

func (s *Schedule) getGatewayRTT() time.Duration {
	if s.gateway != nil {
		if gtw, err := s.gateway.Status.Get(); err == nil && gtw.Rtt != 0 {
			return time.Duration(gtw.Rtt) * time.Millisecond
		}
	}
	return DefaultGatewayRTT
}

func (s *Schedule) GetOption(timestamp uint32, length uint32) (id string, score uint, err error) {
	id = random.String(32)

	s.Lock()
	defer s.Unlock()

	if s.schedule == nil {
		err = ErrScheduleInactive
		return
	}

	item := &scheduledItem{
		id: id,

		timestamp: s.getFullTimestamp(timestamp),
		duration:  length,
	}
	item.time = s.getRealtime(item.timestamp)

	s.items[id] = item

	conflicts := s.schedule.Add(item)

	for _, conflict := range conflicts {
		if i, ok := conflict.(*scheduledItem); ok {
			i.mu.RLock()
			isScheduled := i.payload != nil
			i.mu.RUnlock()
			if isScheduled {
				score += 100
			} else {
				score++
			}
		}
	}

	return
}

func (s *Schedule) Schedule(id string, downlink *router_pb.DownlinkMessage) error {
	s.Lock()
	defer s.Unlock()

	if s.schedule == nil {
		return ErrScheduleInactive
	}

	if item, ok := s.items[id]; ok {
		item.mu.Lock()
		item.payload = downlink
		if lorawan := downlink.GetProtocolConfiguration().GetLorawan(); lorawan != nil {
			var time time.Duration
			if lorawan.Modulation == pb_lorawan.Modulation_LORA {
				// Calculate max ToA
				time, _ = toa.ComputeLoRa(
					uint(len(downlink.Payload)),
					lorawan.DataRate,
					lorawan.CodingRate,
				)
			}
			if lorawan.Modulation == pb_lorawan.Modulation_FSK {
				// Calculate max ToA
				time, _ = toa.ComputeFSK(
					uint(len(downlink.Payload)),
					int(lorawan.BitRate),
				)
			}
			item.duration = uint32(time / 1000)
		}

		copy := &scheduledItem{
			id:      item.id,
			time:    item.time.Add(-1 * (s.getGatewayRTT() + DefaultGatewayBufferTime)),
			payload: item.payload,
		}

		s.ctx.WithField("Identifier", item.id).WithFields(fields.Get(item.payload)).WithField("Remaining", time.Until(copy.time)).Debug("Scheduled Downlink")

		item.mu.Unlock()

		delete(s.items, id)
		s.queue.Add(copy)

		return nil
	}
	return errors.NewErrNotFound(id)
}

func (s *Schedule) Subscribe() <-chan *router_pb.DownlinkMessage {
	s.Lock()
	defer s.Unlock()
	if s.schedule == nil {
		s.init()
		s.subscribers++
	}
	return s.downlink
}

func (s *Schedule) Stop() {
	s.subscribers--
	if s.subscribers < 1 {
		s.Lock()
		defer s.Unlock()
		s.schedule.Destroy()
	}
}

func (s *Schedule) IsActive() bool {
	s.RLock()
	defer s.RUnlock()
	return s.downlink != nil
}

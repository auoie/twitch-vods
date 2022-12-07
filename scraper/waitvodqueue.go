package scraper

import (
	"errors"
	"time"

	"github.com/monitor1379/yagods/maps/treemap"
	"github.com/monitor1379/yagods/utils"
)

func (vod *LiveVod) getWaitVodsKey() *waitVodKey {
	return &waitVodKey{lastInteraction: vod.LastInteraction, streamId: vod.StreamId, startTime: vod.StartTime}
}

type waitVodKey struct {
	lastInteraction time.Time
	streamId        string
	startTime       time.Time
}

type streamIdStartTime struct {
	streamId  string
	startTime time.Time
}

type waitVodsPriorityQueue struct {
	// streamId also acts as a primary key
	streamIdToVod        map[streamIdStartTime]*LiveVod // at most one VOD per (streamId, startTime), there can be multiple VODs per streamer id and per stream id
	lastInteractionToVod *treemap.Map[*waitVodKey, *LiveVod]
}

// This makes it easy to fetch the VOD with the oldest LastInteraction time.
func CreateNewWaitVodsPriorityQueue() *waitVodsPriorityQueue {
	return &waitVodsPriorityQueue{
		streamIdToVod: map[streamIdStartTime]*LiveVod{},
		lastInteractionToVod: treemap.NewWith[*waitVodKey, *LiveVod](func(a, b *waitVodKey) int {
			dif := utils.TimeComparator(a.lastInteraction, b.lastInteraction)
			if dif != 0 {
				return dif
			}
			dif = utils.StringComparator(a.streamId, b.streamId)
			if dif != 0 {
				return dif
			}
			return utils.TimeComparator(a.startTime, b.startTime)
		}),
	}
}

func (vods *waitVodsPriorityQueue) Size() int {
	return vods.lastInteractionToVod.Size()
}

func (vods *waitVodsPriorityQueue) GetStalestStream() (*LiveVod, error) {
	key, vod := vods.lastInteractionToVod.Min()
	if key == nil || vod == nil {
		return nil, errors.New("vods is empty")
	}
	return vod, nil
}

func (vods *waitVodsPriorityQueue) DeleteByStreamIdStartTime(streamId string, startTime time.Time) {
	curVod, ok := vods.streamIdToVod[streamIdStartTime{streamId, startTime}]
	if !ok {
		return
	}
	vods.RemoveVod(curVod)
}

func (vods *waitVodsPriorityQueue) GetByStreamIdStartTime(streamId string, startTime time.Time) (*LiveVod, error) {
	curVod, ok := vods.streamIdToVod[streamIdStartTime{streamId, startTime}]
	if !ok {
		return nil, errors.New("not present")
	}
	return curVod, nil
}

func (vods *waitVodsPriorityQueue) RemoveVod(vod *LiveVod) {
	vods.lastInteractionToVod.Remove(vod.getWaitVodsKey())
	delete(vods.streamIdToVod, streamIdStartTime{streamId: vod.StreamId, startTime: vod.StartTime})
}

// Parameters are the information for the VOD.
// Returns nil error iff new VOD evicts an older VOD.
// In the above case, the returned VOD will be the evicted VOD.
func (vods *waitVodsPriorityQueue) Put(liveVod *LiveVod) {
	vods.DeleteByStreamIdStartTime(liveVod.StreamId, liveVod.StartTime)
	vods.streamIdToVod[streamIdStartTime{streamId: liveVod.StreamId, startTime: liveVod.StartTime}] = liveVod
	vods.lastInteractionToVod.Put(liveVod.getWaitVodsKey(), liveVod)
}

package scraper

import (
	"errors"
	"time"

	"github.com/monitor1379/yagods/maps/treemap"
	"github.com/monitor1379/yagods/utils"
)

func (vod *LiveVod) getWaitVodsKey() *waitVodKey {
	return &waitVodKey{lastInteraction: vod.LastInteraction, streamId: vod.StreamId}
}

type waitVodKey struct {
	lastInteraction time.Time
	streamId        string
}

type waitVodsPriorityQueue struct {
	// streamId also acts as a primary key
	streamIdToVod        map[string]*LiveVod // at most one VOD per stream id, there can be multiple VODs per streamer id
	lastInteractionToVod *treemap.Map[*waitVodKey, *LiveVod]
}

// This makes it easy to fetch the VOD with the oldest LastInteraction time.
func CreateNewWaitVodsPriorityQueue() *waitVodsPriorityQueue {
	return &waitVodsPriorityQueue{
		streamIdToVod: map[string]*LiveVod{},
		lastInteractionToVod: treemap.NewWith[*waitVodKey, *LiveVod](func(a, b *waitVodKey) int {
			dif := utils.TimeComparator(a.lastInteraction, b.lastInteraction)
			if dif != 0 {
				return dif
			}
			return utils.StringComparator(a.streamId, b.streamId)
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

func (vods *waitVodsPriorityQueue) DeleteByStreamId(streamId string) {
	curVod, ok := vods.streamIdToVod[streamId]
	if !ok {
		return
	}
	vods.RemoveVod(curVod)
}

func (vods *waitVodsPriorityQueue) GetByStreamId(streamId string) (*LiveVod, error) {
	curVod, ok := vods.streamIdToVod[streamId]
	if !ok {
		return nil, errors.New("not present")
	}
	return curVod, nil
}

func (vods *waitVodsPriorityQueue) RemoveVod(vod *LiveVod) {
	vods.lastInteractionToVod.Remove(vod.getWaitVodsKey())
	delete(vods.streamIdToVod, vod.StreamId)
}

// Parameters are the information for the VOD.
// Returns nil error iff new VOD evicts an older VOD.
// In the above case, the returned VOD will be the evicted VOD.
func (vods *waitVodsPriorityQueue) Put(liveVod *LiveVod) {
	vods.DeleteByStreamId(liveVod.StreamId)
	vods.streamIdToVod[liveVod.StreamId] = liveVod
	vods.lastInteractionToVod.Put(liveVod.getWaitVodsKey(), liveVod)
}

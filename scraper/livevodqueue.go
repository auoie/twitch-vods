package scraper

import (
	"errors"
	"time"

	"github.com/monitor1379/yagods/maps/treemap"
	"github.com/monitor1379/yagods/utils"
)

func getMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (vod *LiveVod) getLiveVodsKey() *liveVodKey {
	return &liveVodKey{lastUpdated: vod.LastUpdated, streamId: vod.StreamId}
}

type liveVodKey struct {
	lastUpdated time.Time
	streamId    string
}

type liveVodsPriorityQueue struct {
	// streamerId acts as a primary key
	// streamId also acts as a primary key
	streamerIdToVod  map[string]*LiveVod // at most one VOD per streamer id
	streamIdToVod    map[string]*LiveVod // at most one VOD per stream id
	lastUpdatedToVod *treemap.Map[*liveVodKey, *LiveVod]
}

// This makes it easy to fetch the VOD with the oldest LastUpdated time.
func CreateNewLiveVodsPriorityQueue() *liveVodsPriorityQueue {
	return &liveVodsPriorityQueue{
		streamerIdToVod: map[string]*LiveVod{},
		streamIdToVod:   map[string]*LiveVod{},
		lastUpdatedToVod: treemap.NewWith[*liveVodKey, *LiveVod](func(a, b *liveVodKey) int {
			dif := utils.TimeComparator(a.lastUpdated, b.lastUpdated)
			if dif != 0 {
				return dif
			}
			return utils.StringComparator(a.streamId, b.streamId)
		}),
	}
}

func (vods *liveVodsPriorityQueue) Size() int {
	return vods.lastUpdatedToVod.Size()
}

func (vods *liveVodsPriorityQueue) GetStalestStream() (*LiveVod, error) {
	key, vod := vods.lastUpdatedToVod.Min()
	if key == nil || vod == nil {
		return nil, errors.New("vods is empty")
	}
	return vod, nil
}

func (vods *liveVodsPriorityQueue) RemoveVod(vod *LiveVod) {
	vods.lastUpdatedToVod.Remove(vod.getLiveVodsKey())
	delete(vods.streamIdToVod, vod.StreamId)
	delete(vods.streamerIdToVod, vod.StreamerId)
}

// Parameters are the information for the VOD.
// Returns nil error iff new VOD evicts an older VOD.
// In the above case, the returned VOD will be the evicted VOD.
// This code scares me. It's probably buggy.
func (vods *liveVodsPriorityQueue) UpsertVod(
	curTime time.Time,
	data VodDataPoint) (*LiveVod, error) {
	node := data.Node
	streamerId := node.Broadcaster.Id
	streamerLogin := node.Broadcaster.Login
	streamId := node.Id
	startTime := node.CreatedAt
	viewers := node.ViewersCount
	curVod, ok := vods.streamerIdToVod[streamerId] // check if the streamer has an old stream
	if !ok {
		newVod := &LiveVod{
			StreamerId:           streamerId,
			StreamId:             streamId,
			StartTime:            startTime,
			StreamerLoginAtStart: streamerLogin,
			MaxViews:             viewers,
			LastUpdated:          curTime,
		}
		vods.lastUpdatedToVod.Put(newVod.getLiveVodsKey(), newVod)
		vods.streamIdToVod[streamId] = newVod
		vods.streamerIdToVod[streamerId] = newVod
		return nil, errors.New("VOD is new")
	} else if curVod.StartTime != startTime {
		vods.RemoveVod(curVod)
		newVod := &LiveVod{
			StreamerId:           streamerId,
			StreamId:             streamId,
			StartTime:            startTime,
			StreamerLoginAtStart: streamerLogin,
			MaxViews:             viewers,
			LastUpdated:          curTime,
		}
		vods.lastUpdatedToVod.Put(newVod.getLiveVodsKey(), newVod)
		vods.streamIdToVod[streamId] = newVod
		vods.streamerIdToVod[streamerId] = newVod
		return curVod, nil
	} else {
		vods.RemoveVod(curVod)
		curVod.MaxViews = getMax(viewers, curVod.MaxViews)
		curVod.LastUpdated = curTime
		vods.lastUpdatedToVod.Put(curVod.getLiveVodsKey(), curVod)
		vods.streamIdToVod[streamId] = curVod
		vods.streamerIdToVod[streamerId] = curVod
		return nil, errors.New("VOD exists and has been updated")
	}
}

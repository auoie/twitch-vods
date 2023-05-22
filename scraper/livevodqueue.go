package scraper

import (
	"errors"
	"fmt"
	"log"
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
	return &liveVodKey{lastUpdatedUnix: vod.LastUpdatedUnix, streamId: vod.StreamId}
}

type liveVodKey struct {
	lastUpdatedUnix int64
	streamId        string
}

type liveVodsPriorityQueue struct {
	// streamerId acts as a primary key
	streamerIdToVod  map[string]*LiveVod // at most one VOD per streamer id
	lastUpdatedToVod *treemap.Map[*liveVodKey, *LiveVod]
}

// This makes it easy to fetch the VOD with the oldest LastUpdated time.
func CreateNewLiveVodsPriorityQueue() *liveVodsPriorityQueue {
	return &liveVodsPriorityQueue{
		streamerIdToVod: map[string]*LiveVod{},
		lastUpdatedToVod: treemap.NewWith[*liveVodKey, *LiveVod](func(a, b *liveVodKey) int {
			dif := utils.NumberComparator(a.lastUpdatedUnix, b.lastUpdatedUnix)
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
	delete(vods.streamerIdToVod, vod.StreamerId)
}

// Parameters are the information for the VOD.
// Returns nil error iff new VOD evicts an older VOD.
// In the above case, the returned VOD will be the evicted VOD.
func (vods *liveVodsPriorityQueue) UpsertVod(data VodDataPoint) (*LiveVod, error) {
	node := data.Node
	liveVod := &LiveVod{
		StreamerId:           node.UserID,
		StreamId:             node.ID,
		StartTimeUnix:        node.StartedAt.UTC().Unix(),
		StreamerLoginAtStart: node.UserLogin,
		GameIdAtStart:        node.GameID,
		MaxViews:             node.ViewerCount,
		LastUpdatedUnix:      data.ResponseReturnedTimeUnix,
		LastInteractionUnix:  data.ResponseReturnedTimeUnix,
	}
	return vods.UpsertLiveVod(liveVod)
}

// Parameters are the information for the VOD.
// Returns nil error iff new VOD evicts an older VOD.
// In the above case, the returned VOD will be the evicted VOD.
func (vods *liveVodsPriorityQueue) UpsertLiveVod(liveVod *LiveVod) (*LiveVod, error) {
	streamerId := liveVod.StreamerId
	startTimeUnix := liveVod.StartTimeUnix
	viewers := liveVod.MaxViews
	curVod, ok := vods.streamerIdToVod[streamerId] // check if the streamer has an old stream
	if !ok {
		// This is a new stream and streamer doesn't have a stream in the queue
		vods.lastUpdatedToVod.Put(liveVod.getLiveVodsKey(), liveVod)
		vods.streamerIdToVod[streamerId] = liveVod
		return nil, errors.New("VOD is new")
	} else if curVod.StartTimeUnix != startTimeUnix {
		// This is a new stream and streamer has a stream in the queue
		log.Println(fmt.Sprint("curVod.StartTime and startTime: ", time.Unix(curVod.StartTimeUnix, 0).UTC(), " and ", time.Unix(startTimeUnix, 0).UTC()))
		vods.RemoveVod(curVod)
		vods.lastUpdatedToVod.Put(liveVod.getLiveVodsKey(), liveVod)
		vods.streamerIdToVod[streamerId] = liveVod
		return curVod, nil
	} else {
		// This is an old stream
		vods.RemoveVod(curVod)
		curVod.MaxViews = getMax(viewers, curVod.MaxViews)
		curVod.LastUpdatedUnix = liveVod.LastUpdatedUnix
		curVod.LastInteractionUnix = liveVod.LastInteractionUnix
		vods.lastUpdatedToVod.Put(curVod.getLiveVodsKey(), curVod)
		vods.streamerIdToVod[streamerId] = curVod
		return nil, errors.New("VOD exists and has been updated")
	}
}

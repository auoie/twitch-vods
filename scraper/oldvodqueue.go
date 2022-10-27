package scraper

import (
	"errors"

	"github.com/monitor1379/yagods/maps/treemap"
	"github.com/monitor1379/yagods/utils"
)

type oldVodsPriorityQueue struct {
	tree *treemap.Map[*oldVodKey, *LiveVod]
}

type oldVodKey struct {
	maxViews int
	streamId string
}

func (vod *LiveVod) getOldVodKey() *oldVodKey {
	return &oldVodKey{maxViews: vod.MaxViews, streamId: vod.StreamId}
}

func CreateNewOldVodQueue() *oldVodsPriorityQueue {
	return &oldVodsPriorityQueue{
		tree: treemap.NewWith[*oldVodKey, *LiveVod](func(a, b *oldVodKey) int {
			dif := a.maxViews - b.maxViews
			if dif != 0 {
				return dif
			}
			return utils.StringComparator(a.streamId, b.streamId)
		})}
}

func (vods *oldVodsPriorityQueue) Put(vod *LiveVod) {
	vods.tree.Put(vod.getOldVodKey(), vod)
}

func (vods *oldVodsPriorityQueue) Size() int {
	return vods.tree.Size()
}

func (vods *oldVodsPriorityQueue) PopLowViewCount() (*LiveVod, error) {
	_, vod := vods.tree.Min()
	if vod == nil {
		return nil, errors.New("tree is empty")
	}
	vods.tree.Remove(vod.getOldVodKey())
	return vod, nil
}

func (vods *oldVodsPriorityQueue) PopHighViewCount() (*LiveVod, error) {
	_, vod := vods.tree.Max()
	if vod == nil {
		return nil, errors.New("tree is empty")
	}
	vods.tree.Remove(vod.getOldVodKey())
	return vod, nil
}
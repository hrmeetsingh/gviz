package analyzer

import (
	"sort"
	"time"

	"github.com/harmeetsingh/gviz/internal/model"
)

// BuildTree wires parent/child relationships and returns root goroutines.
// Goroutines whose parent ID is -1 or not present in the set become roots.
func BuildTree(goroutines []*model.Goroutine) []*model.Goroutine {
	// Reset children on every call (safe for repeated use).
	for _, g := range goroutines {
		g.Children = nil
	}

	byID := make(map[int64]*model.Goroutine, len(goroutines))
	for _, g := range goroutines {
		byID[g.ID] = g
	}

	var roots []*model.Goroutine
	for _, g := range goroutines {
		if g.ParentID == -1 {
			roots = append(roots, g)
			continue
		}
		parent, ok := byID[g.ParentID]
		if !ok {
			roots = append(roots, g)
			continue
		}
		parent.Children = append(parent.Children, g)
	}

	// Sort for stable, deterministic rendering.
	sortByID(roots)
	for _, g := range goroutines {
		sortByID(g.Children)
	}
	return roots
}

func sortByID(gs []*model.Goroutine) {
	sort.Slice(gs, func(i, j int) bool { return gs[i].ID < gs[j].ID })
}

// ChannelPair represents an inferred send/receive relationship.
type ChannelPair struct {
	SenderID   int64
	ReceiverID int64
}

// InferChannelPairs groups goroutines that are blocked on the same channel
// (one send, one recv) into pairs. Since we can't see channel addresses from
// stack text alone, we pair all senders with all receivers that share the same
// parent (i.e. likely the same channel).
func InferChannelPairs(goroutines []*model.Goroutine) []ChannelPair {
	senders := make(map[int64][]int64)   // parentID → sender IDs
	receivers := make(map[int64][]int64) // parentID → receiver IDs

	for _, g := range goroutines {
		switch g.State {
		case model.StateChanSend:
			senders[g.ParentID] = append(senders[g.ParentID], g.ID)
		case model.StateChanRecv:
			receivers[g.ParentID] = append(receivers[g.ParentID], g.ID)
		}
	}

	var pairs []ChannelPair
	for parentID, senderIDs := range senders {
		recvIDs := receivers[parentID]
		for i, sID := range senderIDs {
			rID := int64(-1)
			if i < len(recvIDs) {
				rID = recvIDs[i]
			}
			pairs = append(pairs, ChannelPair{SenderID: sID, ReceiverID: rID})
		}
	}
	return pairs
}

// Diff computes new and ended goroutine IDs between two lists.
func Diff(prev, curr []*model.Goroutine) (newIDs []int64, endedIDs []int64) {
	prevSet := make(map[int64]bool, len(prev))
	for _, g := range prev {
		prevSet[g.ID] = true
	}
	currSet := make(map[int64]bool, len(curr))
	for _, g := range curr {
		currSet[g.ID] = true
	}

	for _, g := range curr {
		if !prevSet[g.ID] {
			newIDs = append(newIDs, g.ID)
		}
	}
	for _, g := range prev {
		if !currSet[g.ID] {
			endedIDs = append(endedIDs, g.ID)
		}
	}
	return
}

// BuildSnapshot constructs a Snapshot from a current goroutine list and an
// optional previous snapshot (nil for the first snapshot).
func BuildSnapshot(goroutines []*model.Goroutine, prev *model.Snapshot) *model.Snapshot {
	roots := BuildTree(goroutines)

	byID := make(map[int64]*model.Goroutine, len(goroutines))
	for _, g := range goroutines {
		byID[g.ID] = g
	}

	snap := &model.Snapshot{
		At:   time.Now(),
		Roots: roots,
		ByID: byID,
	}

	if prev != nil {
		var prevList []*model.Goroutine
		for _, g := range prev.ByID {
			prevList = append(prevList, g)
		}
		snap.NewIDs, snap.EndedIDs = Diff(prevList, goroutines)
	}

	return snap
}

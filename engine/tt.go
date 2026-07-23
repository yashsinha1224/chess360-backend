package engine

const maxSearchPly = 128

type TTFlag int

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Key      uint64
	Depth    int
	Score    int
	Flag     TTFlag
	BestMove Move
	valid    bool
}

type TranspositionTable struct {
	entries []TTEntry
	mask    uint64
}

func NewTranspositionTable(sizeMB int) *TranspositionTable {
	const bytesPerEntry = 64
	numEntries := (sizeMB * 1024 * 1024) / bytesPerEntry
	numEntries = nextPowerOfTwoTT(numEntries)
	if numEntries < 1 {
		numEntries = 1
	}
	return &TranspositionTable{
		entries: make([]TTEntry, numEntries),
		mask:    uint64(numEntries - 1),
	}
}

func nextPowerOfTwoTT(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func (tt *TranspositionTable) index(key uint64) uint64 {
	return key & tt.mask
}

func (tt *TranspositionTable) Probe(key uint64) (TTEntry, bool) {
	e := tt.entries[tt.index(key)]
	if !e.valid || e.Key != key {
		return TTEntry{}, false
	}
	return e, true
}

func (tt *TranspositionTable) Store(key uint64, depth, score int, flag TTFlag, best Move) {
	idx := tt.index(key)
	existing := tt.entries[idx]
	if existing.valid && existing.Key != key && existing.Depth > depth {
		return
	}
	tt.entries[idx] = TTEntry{
		Key: key, Depth: depth, Score: score, Flag: flag, BestMove: best, valid: true,
	}
}

const mateScoreThreshold = MateScore - maxSearchPly

func scoreToTT(score, ply int) int {
	switch {
	case score >= mateScoreThreshold:
		return score + ply
	case score <= -mateScoreThreshold:
		return score - ply
	default:
		return score
	}
}

func scoreFromTT(score, ply int) int {
	switch {
	case score >= mateScoreThreshold:
		return score - ply
	case score <= -mateScoreThreshold:
		return score + ply
	default:
		return score
	}
}

func isForcedMateScore(score int) bool {
	return score >= mateScoreThreshold || score <= -mateScoreThreshold
}

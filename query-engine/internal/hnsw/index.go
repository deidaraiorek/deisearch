package hnsw

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"

	"gonum.org/v1/gonum/floats"
)

type Index struct {
	dimension int
	m         int
	mMax      int
	efConstruction int
	ef        int
	ml        float64
	nodes     []*Node
	entryPoint int
	mu        sync.RWMutex
}

type Node struct {
	id int
	vector []float32
	level int
	neighbors [][]int
}

type distanceHeap []distancePair

type distancePair struct {
	id int
	distance float64
}

func (h distanceHeap) Len() int { return len(h) }
func (h distanceHeap) Less(i, j int) bool { return h[i].distance < h[j].distance }
func (h distanceHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *distanceHeap) Push(x interface{}) { *h = append(*h, x.(distancePair)) }
func (h *distanceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0:n-1]
	return x
}

func NewIndex(dimension, maxElements int) *Index {
	return &Index{
		dimension: dimension,
		m: 16,
		mMax: 16,
		efConstruction: 200,
		ef: 100,
		ml: 1.0 / math.Log(2.0),
		nodes: make([]*Node, 0, maxElements),
		entryPoint: -1,
	}
}

func (idx *Index) SetEf(ef int) {
	idx.ef = ef
}

func (idx *Index) AddPoint(vector []float32, id int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	level := idx.randomLevel()
	node := &Node{
		id: id,
		vector: vector,
		level: level,
		neighbors: make([][]int, level+1),
	}

	if idx.entryPoint == -1 {
		idx.entryPoint = len(idx.nodes)
		idx.nodes = append(idx.nodes, node)
		return
	}

	nodeIdx := len(idx.nodes)
	idx.nodes = append(idx.nodes, node)

	ep := idx.entryPoint
	for lc := idx.nodes[ep].level; lc > level; lc-- {
		ep = idx.searchLayer(vector, ep, 1, lc)[0].id
	}

	for lc := level; lc >= 0; lc-- {
		candidates := idx.searchLayer(vector, ep, idx.efConstruction, lc)

		m := idx.m
		if lc == 0 {
			m = idx.mMax
		}

		neighbors := idx.selectNeighbors(candidates, m)

		for _, neighbor := range neighbors {
			idx.nodes[nodeIdx].neighbors[lc] = append(idx.nodes[nodeIdx].neighbors[lc], neighbor)
			idx.nodes[neighbor].neighbors[lc] = append(idx.nodes[neighbor].neighbors[lc], nodeIdx)

			if len(idx.nodes[neighbor].neighbors[lc]) > m {
				newNeighbors := idx.pruneConnections(neighbor, lc, m)
				idx.nodes[neighbor].neighbors[lc] = newNeighbors
			}
		}

		if len(candidates) > 0 {
			ep = candidates[0].id
		}
	}

	if level > idx.nodes[idx.entryPoint].level {
		idx.entryPoint = nodeIdx
	}
}

func (idx *Index) Search(query []float32, k int) []int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.entryPoint == -1 {
		return nil
	}

	ep := idx.entryPoint
	for lc := idx.nodes[ep].level; lc > 0; lc-- {
		nearest := idx.searchLayer(query, ep, 1, lc)
		if len(nearest) > 0 {
			ep = nearest[0].id
		}
	}

	candidates := idx.searchLayer(query, ep, idx.ef, 0)

	result := make([]int, 0, k)
	for i := 0; i < len(candidates) && i < k; i++ {
		result = append(result, idx.nodes[candidates[i].id].id)
	}

	return result
}

func (idx *Index) searchLayer(query []float32, ep int, ef int, level int) []distancePair {
	visited := make(map[int]bool)
	candidates := &distanceHeap{}
	w := &distanceHeap{}

	dist := idx.distance(query, idx.nodes[ep].vector)
	heap.Push(candidates, distancePair{ep, dist})
	heap.Push(w, distancePair{ep, -dist})
	visited[ep] = true

	for candidates.Len() > 0 {
		c := heap.Pop(candidates).(distancePair)
		f := (*w)[0].distance

		if c.distance > -f {
			break
		}

		for _, neighbor := range idx.nodes[c.id].neighbors[level] {
			if !visited[neighbor] {
				visited[neighbor] = true

				dist := idx.distance(query, idx.nodes[neighbor].vector)
				f := (*w)[0].distance

				if dist < -f || w.Len() < ef {
					heap.Push(candidates, distancePair{neighbor, dist})
					heap.Push(w, distancePair{neighbor, -dist})

					if w.Len() > ef {
						heap.Pop(w)
					}
				}
			}
		}
	}

	result := make([]distancePair, w.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(w).(distancePair)
		result[i].distance = -result[i].distance
	}

	return result
}

func (idx *Index) selectNeighbors(candidates []distancePair, m int) []int {
	if len(candidates) <= m {
		result := make([]int, len(candidates))
		for i, c := range candidates {
			result[i] = c.id
		}
		return result
	}

	result := make([]int, m)
	for i := 0; i < m; i++ {
		result[i] = candidates[i].id
	}
	return result
}

func (idx *Index) pruneConnections(nodeIdx int, level int, m int) []int {
	neighbors := idx.nodes[nodeIdx].neighbors[level]
	if len(neighbors) <= m {
		return neighbors
	}

	distances := make([]distancePair, len(neighbors))
	for i, n := range neighbors {
		distances[i] = distancePair{
			id: n,
			distance: idx.distance(idx.nodes[nodeIdx].vector, idx.nodes[n].vector),
		}
	}

	h := distanceHeap(distances)
	heap.Init(&h)

	result := make([]int, m)
	for i := 0; i < m; i++ {
		result[i] = heap.Pop(&h).(distancePair).id
	}

	return result
}

func (idx *Index) distance(a, b []float32) float64 {
	aF64 := make([]float64, len(a))
	bF64 := make([]float64, len(b))
	for i := range a {
		aF64[i] = float64(a[i])
		bF64[i] = float64(b[i])
	}
	return 1.0 - floats.Dot(aF64, bF64)
}

func (idx *Index) randomLevel() int {
	level := 0
	for rand.Float64() < idx.ml && level < 16 {
		level++
	}
	return level
}

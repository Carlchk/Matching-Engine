package main

import (
	"container/heap"
	"sync"
)

type Orderbook struct {
	h *ExtendedHeap
	m map[string]*HeapItem
	sync.Mutex

	depth [][2]string
}

func (o *Orderbook) Len() int {
	return o.h.Len()
}

func (o *Orderbook) Push(item HeapItem) (exist bool) {
	o.Lock()
	defer o.Unlock()

	id := item.GetUniqueId()
	if _, ok := o.m[id]; ok {
		return true
	}

	heap.Push(o.h, item)
	o.m[id] = &item
	return false
}

func (o *Orderbook) Get(index int) HeapItem {
	n := o.h.Len()
	if n <= index {
		return nil
	}

	return (*o.h)[index]
}

func (o *Orderbook) Root() HeapItem {
	return o.Get(0)
}

func (o *Orderbook) Remove(uniqId string) HeapItem {
	o.Lock()
	defer o.Unlock()

	old, ok := o.m[uniqId]
	if !ok {
		return nil
	}

	item := heap.Remove(o.h, (*old).GetIndex())
	delete(o.m, uniqId)
	return item.(HeapItem)
}

func (o *Orderbook) clean() {
	o.Lock()
	defer o.Unlock()

	h := make(ExtendedHeap, 0)
	heap.Init(&h)
	o.h = &h
	o.m = make(map[string]*HeapItem)
}

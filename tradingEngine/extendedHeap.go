package main

import (
	"container/heap"
	"github.com/shopspring/decimal"
)

type HeapItem interface {
	SetIndex(index int)
	SetQuantity(quantity decimal.Decimal)
	SetAmount(amount decimal.Decimal)
	Less(item HeapItem) bool
	GetIndex() int
	GetUniqueId() string
	GetPrice() decimal.Decimal
	GetQuantity() decimal.Decimal
	GetCreateTime() int64
	GetOrderSide() OrderSide
	GetPriceType() PriceType
}

type ExtendedHeap []HeapItem

func (heap ExtendedHeap) Len() int { return len(heap) }

func (heap *ExtendedHeap) Pop() interface{} {
	old := *heap
	n := len(old)
	item := old[n-1]
	item.SetIndex(-1)
	*heap = old[0 : n-1]
	return item
}

func (heap *ExtendedHeap) Push(x interface{}) {
	n := len(*heap)
	x.(HeapItem).SetIndex(n)
	*heap = append(*heap, x.(HeapItem))
}

func (heap ExtendedHeap) Less(i, j int) bool {
	return heap[i].Less(heap[j])
}

func (heap ExtendedHeap) Swap(i, j int) {
	heap[i], heap[j] = heap[j], heap[i]
	heap[i].SetIndex(i)
	heap[j].SetIndex(j)
}

func NewOrderBook() *Orderbook {
	h := make(ExtendedHeap, 0)
	heap.Init(&h)

	queue := Orderbook{
		h: &h,
		m: make(map[string]*HeapItem),
	}
	return &queue
}


package main

import (
	"sync"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

// Order book with two red black tree for bids and asks
type Orderbook struct {
	rbt *RedBlackTreeExtended
	m   map[string]*TreeItem
	sync.Mutex

	depth [][2]string
}

func NewOrderBook() *Orderbook {
	ob := Orderbook{
		// rbt: &RedBlackTreeExtended{&rbt.Tree{Comparator: DecimalComparator}},
		rbt: &RedBlackTreeExtended{&rbt.Tree{Comparator: utils.StringComparator}},
		m:   make(map[string]*TreeItem),
	}

	return &ob
}

func (ob *Orderbook) Len() int {
	return ob.rbt.Len()
}

func (ob *Orderbook) Push(item TreeItem) (exist bool) {
	ob.Lock()
	defer ob.Unlock()

	_, exist = ob.m[item.GetUniqueId()]
	if exist {
		return
	}

	ob.rbt.Push(item)
	ob.m[item.GetUniqueId()] = &item
	return
}

func (ob *Orderbook) Top() TreeItem {
	node := (ob.rbt.Root)
	if node == nil {
		return nil
	}
	return node.Value.(TreeItem)
}

func (ob *Orderbook) Remove(qid string) bool {
	ob.Lock()
	defer ob.Unlock()

	item, ok := ob.m[qid]
	if !ok {
		return false
	}

	ob.rbt.Remove((*item).GetUniqueId())
	delete(ob.m, qid)
	return false
}

func (ob *Orderbook) clean() {
	ob.Lock()
	defer ob.Unlock()

	ob.rbt.Clear()
	ob.m = make(map[string]*TreeItem)
}

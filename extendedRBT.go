package main

import (
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
)

type TreeItem interface {
	SetQuantity(quantity decimal.Decimal)
	SetAmount(amount decimal.Decimal)
	GetUniqueId() string
	GetPrice() decimal.Decimal
	GetQuantity() decimal.Decimal
	GetCreateTime() int64
	GetOrderSide() OrderSide
	GetPriceType() PriceType
	GetAmount() decimal.Decimal //订单金额，在市价订单的时候生效，限价单不需要这个字段
}

// Orderbook Comparator of decimal.Decimal
func DecimalComparator(a, b interface{}) int {
	aAsserted := a.(decimal.Decimal)
	bAsserted := b.(decimal.Decimal)
	switch {
	case aAsserted.GreaterThan(bAsserted):
		return 1
	case aAsserted.LessThan(bAsserted):
		return -1
	default:
		return 0
	}
}

type RedBlackTreeExtended struct {
	*rbt.Tree
}

func (t *RedBlackTreeExtended) Len() int { return t.Size() }

func (t *RedBlackTreeExtended) Push(x interface{}) {
	// t.Put(x.(TreeItem).GetAmount(), x.(TreeItem))
	t.Put(x.(TreeItem).GetUniqueId(), x.(TreeItem))

}

func (t *RedBlackTreeExtended) getMinNode(node *rbt.Node) (foundNode *rbt.Node, found bool) {
	if node == nil {
		return nil, false
	}
	if node.Left == nil {
		return node, true
	}
	return t.getMinNode(node.Left)
}

func (t *RedBlackTreeExtended) getMaxNode(node *rbt.Node) (foundNode *rbt.Node, found bool) {
	if node == nil {
		return nil, false
	}
	if node.Right == nil {
		return node, true
	}
	return t.getMaxNode(node.Right)
}

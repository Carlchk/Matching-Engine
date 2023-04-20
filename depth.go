package main

import (
	"time"

	"github.com/shopspring/decimal"
	// "github.com/sirupsen/logrus"
)

func (t *TradePair) GetAskDepth(size int) [][2]string {
	return t.depth(t.AsksOrderbook, size)
}

func (t *TradePair) GetBidDepth(size int) [][2]string {
	return t.depth(t.BidsOrderbook, size)
}

func (t *TradePair) depth(ob *Orderbook, size int) [][2]string {
	ob.Lock()
	defer ob.Unlock()

	max := len(ob.depth)
	if size <= 0 || size > max {
		size = max
	}

	return ob.depth[0:size]
}

func (t *TradePair) depthTicker(ob *Orderbook) {

	ticker := time.NewTicker(time.Duration(50) * time.Millisecond)

	for {
		<-ticker.C
		func() {
			t.w.Lock()
			defer t.w.Unlock()

			ob.Lock()
			defer ob.Unlock()

			ob.depth = [][2]string{}
			depthMap := make(map[string]string)

			if ob.Len() > 0 {
				// Traverse the red-black tree using the iterator function
				it := ob.rbt.Iterator()
				for it.Next() {
					node := it.Value().(TreeItem)
					price := FormatDecimal2String(node.GetPrice(), t.priceDigit)

					// If the price already exists in the hashmap, increment its amount by the current node amount
					if _, ok := depthMap[price]; !ok {
						depthMap[price] = FormatDecimal2String(node.GetQuantity(), t.quantityDigit)
					} else {
						old_qunantity, _ := decimal.NewFromString(depthMap[price])
						depthMap[price] = FormatDecimal2String(old_qunantity.Add(node.GetQuantity()), t.quantityDigit)
					}
				}

				// Convert the hashmap to a 2D array sorted by k
				ob.depth = MapToSortedArr(depthMap, ob.rbt.Root.Value.(TreeItem).GetOrderSide())
			}

		}()
	}
}

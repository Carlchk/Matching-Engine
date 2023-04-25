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

			depthMap := make(map[string]string)
			ob.depth = [][2]string{}

			if ob.h.Len() > 0 {
				for i := 0; i < ob.h.Len(); i++ {
					item := (*ob.h)[i]

					price := FormatDecimal2String(item.GetPrice(), t.priceDigit)

					if _, ok := depthMap[price]; !ok {
						depthMap[price] = FormatDecimal2String(item.GetQuantity(), t.quantityDigit)
					} else {
						old_qunantity, _ := decimal.NewFromString(depthMap[price])
						depthMap[price] = FormatDecimal2String(old_qunantity.Add(item.GetQuantity()), t.quantityDigit)
					}
				}

				ob.depth = MapToSortedArr(depthMap, ob.Root().GetOrderSide())

			}

		}()
	}
}

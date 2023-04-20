package main

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type TradeResult struct {
	Symbol        string          `json:"symbol"`
	AskOrderId    string          `json:"ask_order_id"`
	BidOrderId    string          `json:"bid_order_id"`
	TradeQuantity decimal.Decimal `json:"trade_quantity"`
	TradePrice    decimal.Decimal `json:"trade_price"`
	TradeAmount   decimal.Decimal `json:"trade_amount"`
	TradeTime     int64           `json:"trade_time"`
}

type TradePair struct {
	Symbol         string
	ChTradeResult  chan TradeResult
	ChNewOrder     chan TreeItem
	ChCancelResult chan string

	priceDigit    int
	quantityDigit int
	miniTradeQty  decimal.Decimal
	latestPrice   decimal.Decimal

	BidsOrderbook *Orderbook
	AsksOrderbook *Orderbook

	w sync.Mutex
}

func NewTradePair(symbol string, priceDigit, quantityDigit int) *TradePair {
	t := &TradePair{
		Symbol:         symbol,
		ChTradeResult:  make(chan TradeResult, 10),
		ChNewOrder:     make(chan TreeItem),
		ChCancelResult: make(chan string, 10),

		priceDigit:    priceDigit,
		quantityDigit: quantityDigit,
		miniTradeQty:  decimal.New(1, int32(-quantityDigit)),

		AsksOrderbook: NewOrderBook(),
		BidsOrderbook: NewOrderBook(),
	}

	// todo: depth handling

	go t.depthTicker(t.AsksOrderbook)
	go t.depthTicker(t.BidsOrderbook)

	go t.matching()
	return t
}

func (t *TradePair) matching() {

	for {
		select {
		case newOrder := <-t.ChNewOrder:
			go t.handlerNewOrder(newOrder)
		default:
			t.handlerLimitOrder()
		}

	}

}

func (t *TradePair) handlerNewOrder(newOrder TreeItem) {
	t.w.Lock()
	defer t.w.Unlock()

	if newOrder.GetPriceType() == PriceTypeLimit {
		if newOrder.GetOrderSide() == OrderSideSell {
			t.AsksOrderbook.Push(newOrder)
		} else {
			t.BidsOrderbook.Push(newOrder)
		}
	}
}

func (t *TradePair) handlerLimitOrder() {
	ok := func() bool {
		t.w.Lock()
		defer t.w.Unlock()

		if t.AsksOrderbook == nil || t.BidsOrderbook == nil {
			return false
		}

		if t.AsksOrderbook.Len() == 0 || t.BidsOrderbook.Len() == 0 {
			return false
		}

		askTop := t.AsksOrderbook.Top()
		bidTop := t.BidsOrderbook.Top()

		defer func() {
			if askTop.GetQuantity().Equal(decimal.Zero) {
				t.AsksOrderbook.Remove(askTop.GetUniqueId())
			}
			if bidTop.GetQuantity().Equal(decimal.Zero) {
				t.BidsOrderbook.Remove(bidTop.GetUniqueId())
			}
		}()

		if bidTop.GetPrice().Cmp(askTop.GetPrice()) >= 0 {
			curTradeQty := decimal.Zero
			curTradePrice := decimal.Zero
			if bidTop.GetQuantity().Cmp(askTop.GetQuantity()) >= 0 {
				curTradeQty = askTop.GetQuantity()
			} else if bidTop.GetQuantity().Cmp(askTop.GetQuantity()) == -1 {
				curTradeQty = bidTop.GetQuantity()
			}
			askTop.SetQuantity(askTop.GetQuantity().Sub(curTradeQty))
			bidTop.SetQuantity(bidTop.GetQuantity().Sub(curTradeQty))

			if askTop.GetCreateTime() >= bidTop.GetCreateTime() {
				curTradePrice = bidTop.GetPrice()
			} else {
				curTradePrice = askTop.GetPrice()
			}

			t.sendTradeResultNotify(askTop, bidTop, curTradePrice, curTradeQty, "")
			return true
		} else {
			return false
		}

	}()

	if !ok {
		time.Sleep(time.Duration(60) * time.Millisecond)
	} else {
		if Debug {
			time.Sleep(time.Second * time.Duration(1))
		}
	}
}

func (t *TradePair) sendTradeResultNotify(ask, bid TreeItem, price, tradeQty decimal.Decimal, market_done string) {
	tradelog := TradeResult{}
	tradelog.Symbol = t.Symbol
	tradelog.AskOrderId = ask.GetUniqueId()
	tradelog.BidOrderId = bid.GetUniqueId()
	tradelog.TradeQuantity = tradeQty
	tradelog.TradePrice = price
	tradelog.TradeTime = time.Now().UnixNano()
	tradelog.TradeAmount = tradeQty.Mul(price)
	t.latestPrice = price

	if Debug {
		logrus.Infof("%s tradelog: %+v", t.Symbol, tradelog)
	}

	t.ChTradeResult <- tradelog
}

func (t *TradePair) AskLen() int {
	t.w.Lock()
	defer t.w.Unlock()

	return t.AsksOrderbook.Len()
}

func (t *TradePair) BidLen() int {
	t.w.Lock()
	defer t.w.Unlock()

	return t.BidsOrderbook.Len()
}

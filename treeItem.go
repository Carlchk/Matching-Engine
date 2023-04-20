package main

import (
	"github.com/shopspring/decimal"
)

type Order struct {
	orderId    string
	priceType  PriceType
	price      decimal.Decimal
	quantity   decimal.Decimal
	createTime int64
	amount     decimal.Decimal
}

func (o *Order) SetQuantity(qnt decimal.Decimal) {
	o.quantity = qnt
}

func (o *Order) SetAmount(amount decimal.Decimal) {
	o.amount = amount
}

func (o *Order) GetUniqueId() string {
	return o.orderId
}

func (o *Order) GetPrice() decimal.Decimal {
	return o.price
}

func (o *Order) GetQuantity() decimal.Decimal {
	return o.quantity
}

func (o *Order) GetCreateTime() int64 {
	return o.createTime
}

func (o *Order) GetPriceType() PriceType {
	return o.priceType
}
func (o *Order) GetAmount() decimal.Decimal {
	return o.amount
}

type AskItem struct {
	Order
}

func (a *AskItem) GetOrderSide() OrderSide {
	return OrderSideSell
}

type BidItem struct {
	Order
}

func (a *BidItem) GetOrderSide() OrderSide {
	return OrderSideBuy
}



func NewAskItem(pt PriceType, uniqId string, price, quantity, amount decimal.Decimal, createTime int64) *AskItem {
	return &AskItem{
		Order: Order{
			orderId:    uniqId,
			price:      price,
			quantity:   quantity,
			createTime: createTime,
			priceType:  pt,
			amount:     amount,
		},
	}
}

func NewBidItem(pt PriceType, uniqId string, price, quantity, amount decimal.Decimal, createTime int64) *BidItem {
	return &BidItem{
		Order: Order{
			orderId:    uniqId,
			price:      price,
			quantity:   quantity,
			createTime: createTime,
			priceType:  pt,
			amount:     amount,
		}}
}

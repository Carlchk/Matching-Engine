package main

import (
	"fmt"
	"github.com/shopspring/decimal"
)

func string2decimal(a string) decimal.Decimal {
	d, _ := decimal.NewFromString(a)
	return d
}

func FormatDecimal2String(d decimal.Decimal, digit int) string {
	f, _ := d.Float64()
	format := "%." + fmt.Sprintf("%d", digit) + "f"
	return fmt.Sprintf(format, f)
}

func (t *TradePair) Price2String(price decimal.Decimal) string {
	return FormatDecimal2String(price, t.priceDigit)
}

func (t *TradePair) Qty2String(qty decimal.Decimal) string {
	return FormatDecimal2String(qty, t.quantityDigit)
}

func merge(left, right []string, asc_desc string) []string {
	result := []string{}

	for len(left) > 0 || len(right) > 0 {
		if len(left) > 0 && len(right) > 0 {
			leftDec, _ := decimal.NewFromString(left[0])
			rightDec, _ := decimal.NewFromString(right[0])
			if (asc_desc == "asc" && leftDec.Cmp(rightDec) == -1) || (asc_desc == "desc" && leftDec.Cmp(rightDec) == 1) {
				result = append(result, left[0])
				left = left[1:]
			} else {
				result = append(result, right[0])
				right = right[1:]
			}
		} else if len(left) > 0 {
			result = append(result, left[0])
			left = left[1:]
		} else if len(right) > 0 {
			result = append(result, right[0])
			right = right[1:]
		}
	}

	return result
}

func mergeSort(nums []string, asc_desc string) []string {
	if len(nums) <= 1 {
		return nums
	}

	mid := len(nums) / 2
	left := nums[:mid]
	right := nums[mid:]

	left = mergeSort(left, asc_desc)
	right = mergeSort(right, asc_desc)

	return merge(left, right, asc_desc)
}

func MapToSortedArr(m map[string]string, ask_bid OrderSide) [][2]string {
	res := [][2]string{}
	keys := []string{}
	for k, _ := range m {
		keys = append(keys, k)
	}

	if ask_bid == OrderSideSell {
		keys = mergeSort(keys, "asc")
	} else {
		keys = mergeSort(keys, "desc")
	}

	for _, k := range keys {
		res = append(res, [2]string{k, m[k]})
	}
	return res
}

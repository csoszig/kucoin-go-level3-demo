package order_book

import (
	"sort"

	"github.com/Kucoin/kucoin-go-level3-demo/helper"
	"github.com/shopspring/decimal"
)

//[3]string{"orderId", "price", "size"}
//ask Sort price from low to high
func insertLevel3AskOrderBook(data Level3AsksOrderBook, el Level3OrderBookItem) Level3AsksOrderBook {
	if el[2] == "0" { //hidden order
		return data
	}

	index := sort.Search(len(data), func(i int) bool {
		aF, err := decimal.NewFromString(data[i][1])
		if err != nil {
			panic("format failed: " + data[i][1])
		}
		bF, _ := decimal.NewFromString(el[1])
		if err != nil {
			panic("format failed: " + el[1])
		}

		return aF.GreaterThan(bF)
	})

	data = append(data, [3]string{})
	copy(data[index+1:], data[index:])
	data[index] = el
	return data
}

//[3]string{"orderId", "price", "size"}
//bid Sort price from high to low
func insertLevel3BidOrderBook(data Level3BidsOrderBook, el Level3OrderBookItem) Level3BidsOrderBook {
	if el[2] == "0" { //hidden order
		return data
	}

	index := sort.Search(len(data), func(i int) bool {
		aF, err := decimal.NewFromString(data[i][1])
		if err != nil {
			panic("format failed: " + data[i][1])
		}
		bF, _ := decimal.NewFromString(el[1])
		if err != nil {
			panic("format failed: " + el[1])
		}

		return aF.LessThan(bF)
	})

	data = append(data, [3]string{})
	copy(data[index+1:], data[index:])
	data[index] = el
	return data
}

//[3]string{"orderId", "price", "size"}
func deleteOrderFromLevel3OrderBook(data [][3]string, orderId string) [][3]string {
	for index, item := range data {
		if orderId == item[0] {
			return append(data[:index], data[index+1:]...)
		}
	}

	return data
}

//[3]string{"orderId", "price", "size"}
func updateOrderFromLevel3OrderBook(data [][3]string, makerOrderId string, size string) [][3]string {
	for index, item := range data {
		if makerOrderId == item[0] {
			newSize, err := helper.Bcsub(item[2], size)
			if err != nil {
				panic(err)
			}

			//The reason why the order size with 0 will be removed is because the iceberg order will receive the open message again after the match.
			if err := helper.FloatDiffFromString(newSize, "0"); err == nil {
				return append(data[:index], data[index+1:]...)
			}

			ret := append(data[:index], [3]string{makerOrderId, item[1], newSize})
			return append(ret, data[index+1:]...)
		}
	}

	return data
}

func changeOrderFromLevel3OrderBook(data [][3]string, makerOrderId string, size string) [][3]string {
	for index, item := range data {
		if makerOrderId == item[0] {
			ret := append(data[:index], [3]string{makerOrderId, item[1], size})
			return append(ret, data[index+1:]...)
		}
	}

	return data
}

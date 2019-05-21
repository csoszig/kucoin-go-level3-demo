package order_book

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/Kucoin/kucoin-go-level3-demo/helper"
	"github.com/Kucoin/kucoin-go-sdk"
)

type Level3OrderBook struct {
	apiService *kucoin.ApiService
	symbol     string
	lock       *sync.RWMutex
	Messages   chan json.RawMessage

	fullOrderBook *FullOrderBookModel
}

func NewLevel3OrderBook(apiService *kucoin.ApiService, symbol string) *Level3OrderBook {
	l3book := &Level3OrderBook{
		apiService: apiService,
		symbol:     symbol,
		lock:       &sync.RWMutex{},
		Messages:   make(chan json.RawMessage, 100),
	}

	return l3book
}

func (l3book *Level3OrderBook) Symbol() string {
	return l3book.symbol
}

func (l3book *Level3OrderBook) resetOrderBook() {
	l3book.lock.Lock()
	l3book.fullOrderBook = &FullOrderBookModel{}
	l3book.lock.Unlock()
}

func (l3book *Level3OrderBook) ReloadOrderBook() {
	defer func() {
		if r := recover(); r != nil {
			helper.Error("ReloadOrderBook panic: %v", r)
			l3book.ReloadOrderBook()
		}
	}()

	helper.Info("symbol: %s start ReloadOrderBook", l3book.symbol)
	l3book.resetOrderBook()

	l3book.playback()

	for msg := range l3book.Messages {
		l3Data, err := NewLevel3StreamDataModel(msg)
		if err != nil {
			panic(err)
		}
		l3book.updateFromStream(l3Data)
	}
}

func (l3book *Level3OrderBook) playback() {
	helper.Warn("prepare playback...")

	const tempMsgChanMaxLen = 50
	tempMsgChan := make(chan *Level3StreamDataModel, tempMsgChanMaxLen)
	tempMsgSequenceMap := make(map[string]bool)
	var fullOrderBook *FullOrderBookModel

	for msg := range l3book.Messages {
		l3Data, err := NewLevel3StreamDataModel(msg)
		if err != nil {
			panic(err)
		}

		tempMsgChan <- l3Data
		tempMsgSequenceMap[l3Data.Sequence] = true

		if len(tempMsgChan) > 5 {
			if fullOrderBook == nil {
				fullOrderBook, err = l3book.getAtomicFullOrderBook()
				if err != nil {
					continue
				}
			}

			if fullOrderBook.Sequence <= l3Data.Sequence { //string camp
				if _, ok := tempMsgSequenceMap[fullOrderBook.Sequence]; ok {
					helper.Warn("sequence match, start playback, tempMsgChan: %d", len(tempMsgChan))

					l3book.lock.Lock()
					l3book.fullOrderBook = fullOrderBook
					l3book.lock.Unlock()

					n := len(tempMsgChan)
					for i := 0; i < n; i++ {
						l3book.updateFromStream(<-tempMsgChan)
					}

					helper.Warn("finish playback.")
					break
				} else {
					fullOrderBook = nil
				}
			}

			if len(tempMsgChan) > tempMsgChanMaxLen-5 {
				panic("playback failed, tempMsgChan is too long, retry...")
			}
		}
	}
}

func (l3book *Level3OrderBook) updateFromStream(msg *Level3StreamDataModel) {
	//time.Now().UnixNano()
	helper.Info("msg: %s", string(msg.GetRawMessage()))

	l3book.lock.Lock()
	defer l3book.lock.Unlock()

	skip, err := l3book.updateSequence(msg)
	if err != nil {
		panic(err)
	}

	if !skip {
		l3book.updateOrderBook(msg)
	}
}

func (l3book *Level3OrderBook) updateSequence(msg *Level3StreamDataModel) (bool, error) {
	fullOrderBookSequenceValue, err := helper.Uint64FromString(l3book.fullOrderBook.Sequence)
	if err != nil {
		panic(err)
	}

	msgSequenceValue, err := helper.Uint64FromString(msg.Sequence)
	if err != nil {
		panic(err)
	}

	if fullOrderBookSequenceValue+1 > msgSequenceValue {
		return true, nil
	}

	if fullOrderBookSequenceValue+1 != msgSequenceValue {
		return false, errors.New(fmt.Sprintf(
			"currentSequence: %s, msgSequence: %s, the sequence is not continuous.",
			l3book.fullOrderBook.Sequence,
			msg.Sequence,
		))
	}

	l3book.fullOrderBook.Sequence = msg.Sequence

	return false, nil
}

func (l3book *Level3OrderBook) updateOrderBook(msg *Level3StreamDataModel) {
	//[3]string{"orderId", "price", "size"}
	//var item = [3]string{msg.OrderId, msg.Price, msg.Size}
	switch msg.Side {
	case BuySide:
	case SellSide:
	default:
		panic("error side: " + msg.Side)
	}

	switch msg.Type {
	case Level3MessageReceivedType:
		data := &Level3StreamDataReceivedModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

	case Level3MessageOpenType:
		data := &Level3StreamDataOpenModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		switch data.Side {
		case BuySide:
			l3book.fullOrderBook.Bids = insertLevel3BidOrderBook(
				l3book.fullOrderBook.Bids,
				[3]string{data.OrderId, data.Price, data.Size},
			)

		case SellSide:
			l3book.fullOrderBook.Asks = insertLevel3AskOrderBook(
				l3book.fullOrderBook.Asks,
				[3]string{data.OrderId, data.Price, data.Size},
			)
		}

	case Level3MessageDoneType:
		data := &Level3StreamDataDoneModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		switch data.Side {
		case BuySide:
			l3book.fullOrderBook.Bids = deleteOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Bids,
				data.OrderId,
			)

		case SellSide:
			l3book.fullOrderBook.Asks = deleteOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Asks,
				data.OrderId,
			)
		}

	case Level3MessageMatchType:
		data := &Level3StreamDataMatchModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		switch data.Side {
		case BuySide:
			l3book.fullOrderBook.Asks = updateOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Asks,
				data.MakerOrderId,
				data.Size,
			)

		case SellSide:
			l3book.fullOrderBook.Bids = updateOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Bids,
				data.MakerOrderId,
				data.Size,
			)
		}

	case Level3MessageChangeType:
		data := &Level3StreamDataChangeModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		switch data.Side {
		case BuySide:
			l3book.fullOrderBook.Bids = changeOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Bids,
				data.OrderId,
				data.NewSize,
			)

		case SellSide:
			l3book.fullOrderBook.Asks = changeOrderFromLevel3OrderBook(
				l3book.fullOrderBook.Asks,
				data.OrderId,
				data.NewSize,
			)
		}

	default:
		panic("error msg type: " + msg.Type)
	}
}

func (l3book *Level3OrderBook) getAtomicFullOrderBook() (*FullOrderBookModel, error) {
	rsp, err := l3book.apiService.AtomicFullOrderBook(l3book.symbol)
	if err != nil {
		return nil, err
	}

	c := &FullOrderBookModel{}
	if err := rsp.ReadData(c); err != nil {
		return nil, err
	}

	if c.Sequence == "" {
		return nil, errors.New("empty key sequence")
	}

	return c, nil
}

func (l3book *Level3OrderBook) SnapshotBytes() ([]byte, error) {
	l3book.lock.RLock()
	data, err := json.Marshal(*l3book.fullOrderBook)
	l3book.lock.RUnlock()
	if err != nil {
		return nil, err
	}

	return data, nil
}

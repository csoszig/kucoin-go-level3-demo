package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/JetBlink/orderbook/base"
	"github.com/JetBlink/orderbook/level3"
	"github.com/Kucoin/kucoin-go-level3-demo/helper"
	"github.com/Kucoin/kucoin-go-level3-demo/level3stream"
	"github.com/Kucoin/kucoin-go-level3-demo/log"
	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/shopspring/decimal"
)

type Builder struct {
	apiService *kucoin.ApiService
	symbol     string
	lock       *sync.RWMutex
	Messages   chan json.RawMessage

	fullOrderBook *level3.OrderBook
}

func NewBuilder(apiService *kucoin.ApiService, symbol string) *Builder {
	return &Builder{
		apiService: apiService,
		symbol:     symbol,
		lock:       &sync.RWMutex{},
		Messages:   make(chan json.RawMessage, helper.MaxMsgChanLen),
	}
}

func (b *Builder) resetOrderBook() {
	b.lock.Lock()
	b.fullOrderBook = level3.NewOrderBook()
	b.lock.Unlock()
}

func (b *Builder) ReloadOrderBook() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("ReloadOrderBook panic : %v", r)
			b.ReloadOrderBook()
		}
	}()

	log.Warn("start running ReloadOrderBook, symbol: %s", b.symbol)
	b.resetOrderBook()

	b.playback()

	for msg := range b.Messages {
		l3Data, err := level3stream.NewStreamDataModel(msg)
		if err != nil {
			panic(err)
		}
		b.updateFromStream(l3Data)
	}
}

func (b *Builder) playback() {
	log.Warn("prepare playback...")

	const tempMsgChanMaxLen = 200
	tempMsgChan := make(chan *level3stream.StreamDataModel, tempMsgChanMaxLen)
	firstSequence := ""
	var fullOrderBook *DepthResponse

	for msg := range b.Messages {
		l3Data, err := level3stream.NewStreamDataModel(msg)
		if err != nil {
			panic(err)
		}

		tempMsgChan <- l3Data

		if firstSequence == "" {
			firstSequence = l3Data.Sequence
			log.Error("firstSequence: %s", firstSequence)
		}

		if len(tempMsgChan) > 5 {
			if fullOrderBook == nil {
				log.Warn("start %s GetAtomicFullOrderBook", b.symbol)
				fullOrderBook, err = b.GetAtomicFullOrderBook()
				if err != nil {
					continue
				}
				log.Error("finish GetAtomicFullOrderBook, Sequence: %s", fullOrderBook.Sequence)
			}

			if fullOrderBook != nil && fullOrderBook.Sequence < firstSequence {
				log.Error("GetAtomicFullOrderBook Sequence: %s is too little", fullOrderBook.Sequence)
				fullOrderBook = nil
				continue
			}

			if fullOrderBook != nil && fullOrderBook.Sequence <= l3Data.Sequence { //string camp
				log.Warn("sequence match, start playback, tempMsgChan: %d", len(tempMsgChan))

				b.lock.Lock()
				b.AddDepthToOrderBook(fullOrderBook)
				b.lock.Unlock()

				n := len(tempMsgChan)
				for i := 0; i < n; i++ {
					b.updateFromStream(<-tempMsgChan)
				}

				log.Warn("finish playback.")
				break
			}

			if len(tempMsgChan) > tempMsgChanMaxLen-5 {
				panic("playback failed, tempMsgChan is too long, retry...")
			}
		}
	}
}

func (b *Builder) AddDepthToOrderBook(depth *DepthResponse) {
	b.fullOrderBook.Sequence = helper.ParseUint64OrPanic(depth.Sequence)

	for index, elem := range depth.Asks {
		order, err := level3.NewOrder(elem[0], base.AskSide, elem[1], elem[2], uint64(index), nil)
		if err != nil {
			panic(err)
		}

		if err := b.fullOrderBook.AddOrder(order); err != nil {
			panic(err)
		}
	}

	for index, elem := range depth.Bids {
		order, err := level3.NewOrder(elem[0], base.BidSide, elem[1], elem[2], uint64(index), nil)
		if err != nil {
			panic(err)
		}

		if err := b.fullOrderBook.AddOrder(order); err != nil {
			panic(err)
		}
	}
}

func (b *Builder) updateFromStream(msg *level3stream.StreamDataModel) {
	//log.Info("msg: %s", string(msg.GetRawMessage()))
	b.lock.Lock()
	defer b.lock.Unlock()

	skip, err := b.updateSequence(msg)
	if err != nil {
		panic(err)
	}

	if !skip {
		b.updateOrderBook(msg)
	}
}

func (b *Builder) updateSequence(msg *level3stream.StreamDataModel) (bool, error) {
	fullOrderBookSequenceValue := b.fullOrderBook.Sequence
	msgSequenceValue := helper.ParseUint64OrPanic(msg.Sequence)

	if fullOrderBookSequenceValue+1 > msgSequenceValue {
		return true, nil
	}

	if fullOrderBookSequenceValue+1 != msgSequenceValue {
		return false, errors.New(fmt.Sprintf(
			"currentSequence: %d, msgSequence: %s, the sequence is not continuous, current chanLen: %d",
			b.fullOrderBook.Sequence,
			msg.Sequence,
			len(b.Messages),
		))
	}

	b.fullOrderBook.Sequence = msgSequenceValue

	return false, nil
}

func (b *Builder) updateOrderBook(msg *level3stream.StreamDataModel) {
	side := ""
	switch msg.Side {
	case level3stream.SellSide:
		side = base.AskSide
	case level3stream.BuySide:
		side = base.BidSide
	default:
		panic("error side: " + msg.Side)
	}

	switch msg.Type {
	case level3stream.MessageReceivedType:

	case level3stream.MessageOpenType:
		data := &level3stream.StreamDataOpenModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		if data.Price == "" || data.Size == "0" {
			return
		}

		order, err := level3.NewOrder(data.OrderId, side, data.Price, data.Size, helper.ParseUint64OrPanic(data.Time), nil)
		if err != nil {
			log.Error(string(msg.GetRawMessage()))
			panic(err)
		}
		if err := b.fullOrderBook.AddOrder(order); err != nil {
			panic(err)
		}

	case level3stream.MessageDoneType:
		data := &level3stream.StreamDataDoneModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		if err := b.fullOrderBook.RemoveByOrderId(data.OrderId); err != nil {
			panic(err)
		}

	case level3stream.MessageMatchType:
		data := &level3stream.StreamDataMatchModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		sizeValue, err := decimal.NewFromString(data.Size)
		if err != nil {
			panic(err)
		}
		if err := b.fullOrderBook.MatchOrder(data.MakerOrderId, sizeValue); err != nil {
			panic(err)
		}

	case level3stream.MessageChangeType:
		data := &level3stream.StreamDataChangeModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		sizeValue, err := decimal.NewFromString(data.NewSize)
		if err != nil {
			panic(err)
		}
		if err := b.fullOrderBook.ChangeOrder(data.OrderId, sizeValue); err != nil {
			panic(err)
		}

	default:
		panic("error msg type: " + msg.Type)
	}
}

func (b *Builder) Snapshot() (*FullOrderBook, error) {
	data, err := b.SnapshotBytes()
	if err != nil {
		return nil, err
	}

	ret := &FullOrderBook{}
	if err := json.Unmarshal(data, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (b *Builder) SnapshotBytes() ([]byte, error) {
	b.lock.RLock()
	data, err := json.Marshal(b.fullOrderBook)
	b.lock.RUnlock()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (b *Builder) GetPartOrderBook(number int) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("GetPartOrderBook panic : %v", r)
		}
	}()

	b.lock.RLock()
	defer b.lock.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"sequence":   b.fullOrderBook.Sequence,
		base.AskSide: b.fullOrderBook.GetPartOrderBookBySide(base.AskSide, number),
		base.BidSide: b.fullOrderBook.GetPartOrderBookBySide(base.BidSide, number),
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}

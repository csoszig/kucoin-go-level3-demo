package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/JetBlink/order_book/level3"
	"github.com/Kucoin/kucoin-go-level3-demo/helper"
	"github.com/Kucoin/kucoin-go-level3-demo/level3stream"
	"github.com/Kucoin/kucoin-go-level3-demo/log"
	"github.com/Kucoin/kucoin-go-sdk"
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
			//挂了就重试
			log.Error("ReloadOrderBook panic : %v", r)
			b.ReloadOrderBook()
		}
	}()

	log.Warn("start running ReloadOrderBook, symbol: %s", b.symbol)
	//重置状态
	b.resetOrderBook()

	//重放
	b.playback()

	for msg := range b.Messages {
		l3Data, err := level3stream.NewStreamDataModel(msg)
		if err != nil {
			panic(err)
		}
		b.updateFromStream(l3Data)

		//todo 增加对每条消息time 的时间判断看在多少范围内 同时增加对时间进行比对监控消息更新性能
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
				log.Warn("开始获取 %s 全量数据", b.symbol)
				fullOrderBook, err = b.GetAtomicFullOrderBook()
				if err != nil {
					continue
				}
				log.Error("获取到全量数据, Sequence: %s", fullOrderBook.Sequence)
			}

			//如果 小于 最开始的，则全量太小需要抛弃
			if fullOrderBook != nil && fullOrderBook.Sequence < firstSequence {
				log.Error("获取 %s 全量数据太小", fullOrderBook.Sequence)
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

			if len(tempMsgChan) > tempMsgChanMaxLen-5 { //防止 chan 堵塞
				panic("playback failed, tempMsgChan is too long, retry...")
			}
		}
	}
}

func (b *Builder) AddDepthToOrderBook(depth *DepthResponse) {
	b.fullOrderBook.Sequence = helper.ParseUint64OrPanic(depth.Sequence)

	for index, elem := range depth.Asks {
		order, err := level3.NewOrder(elem[0], elem[1], elem[2], uint64(index))
		if err != nil {
			panic(err)
		}

		b.fullOrderBook.AddOrder(level3.AskSide, order)
	}

	for index, elem := range depth.Bids {
		order, err := level3.NewOrder(elem[0], elem[1], elem[2], uint64(index))
		if err != nil {
			panic(err)
		}

		b.fullOrderBook.AddOrder(level3.BidSide, order)
	}
}

func (b *Builder) updateFromStream(msg *level3stream.StreamDataModel) {
	//time.Now().UnixNano()
	//log.Info("msg: %s", string(msg.GetRawMessage()))

	//获取写锁
	b.lock.Lock()
	//解除写锁
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

	//当前100, 101 > 100 跳过，即100需要跳过
	if fullOrderBookSequenceValue+1 > msgSequenceValue {
		return true, nil
	}

	//不连续
	if fullOrderBookSequenceValue+1 != msgSequenceValue {
		return false, errors.New(fmt.Sprintf(
			"currentSequence: %d, msgSequence: %s, the sequence is not continuous, 当前chanLen: %d",
			b.fullOrderBook.Sequence,
			msg.Sequence,
			len(b.Messages),
		))
	}

	//更新
	//!!! sequence 需要更新，通过判断 sequence 是否自增来校验数据完整性，否则重放数据。
	b.fullOrderBook.Sequence = msgSequenceValue

	return false, nil
}

//todo 大单特别注意
func (b *Builder) updateOrderBook(msg *level3stream.StreamDataModel) {
	//[3]string{"orderId", "price", "size"}
	//var item = [3]string{msg.OrderId, msg.Price, msg.Size}

	//统一处理交易方向
	side := ""
	matchSide := ""
	switch msg.Side {
	case level3stream.SellSide: //卖单
		side = level3.AskSide
		//!!! 卖单 更新 买盘 (maker 是买盘)
		matchSide = level3.BidSide
	case level3stream.BuySide: //买单
		side = level3.BidSide
		//!!! 买单 更新 卖盘 (maker 是卖盘)
		matchSide = level3.AskSide
	default:
		panic("错误的side: " + msg.Side)
	}

	switch msg.Type {
	case level3stream.MessageReceivedType: //不影响买卖盘，暂时不进行处理，以后可以作为分析 todo
		//data := &level3.StreamDataReceivedModel{}
		//if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
		//	panic(err)
		//}

		//if data.OrderType == level3.MarketOrderType {
		//	log.Warn("市价单: " + string(msg.GetRawMessage()))
		//}

	case level3stream.MessageOpenType:
		data := &level3stream.StreamDataOpenModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		if data.Price == "" || data.Size == "0" {
			return
		}

		order, err := level3.NewOrder(data.OrderId, data.Price, data.Size, helper.ParseUint64OrPanic(data.Time))
		if err != nil {
			log.Error(string(msg.GetRawMessage()))
			panic(err)
		}
		b.fullOrderBook.AddOrder(side, order)

	case level3stream.MessageDoneType: //已完成 从买卖盘中去掉 todo 后续可以分析撤单和成交的单情况
		data := &level3stream.StreamDataDoneModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}
		//if data.Reason == Level3MessageDoneFilled {
		//	log.Warn("跟踪到订单 " + Level3MessageDoneFilled)
		//}

		b.fullOrderBook.RemoveByOrderId(side, data.OrderId)

	case level3stream.MessageMatchType:
		data := &level3stream.StreamDataMatchModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		if err := b.fullOrderBook.MatchOrder(matchSide, data.MakerOrderId, data.Size); err != nil {
			panic(err)
		}

	case level3stream.MessageChangeType:
		//只有 DC 的 stp 才会触发 change 消息
		data := &level3stream.StreamDataChangeModel{}
		if err := json.Unmarshal(msg.GetRawMessage(), data); err != nil {
			panic(err)
		}

		//为了醒目
		//log.Error("收到 #change# 消息: ", string(msg.GetRawMessage()))

		if err := b.fullOrderBook.ChangeOrder(side, data.OrderId, data.NewSize); err != nil {
			panic(err)
		}

	default:
		panic("错误的 msg type: " + msg.Type)
	}
}

//获取买卖盘的快照
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
	////获取读锁
	b.lock.RLock()
	data, err := json.Marshal(b.fullOrderBook)
	b.lock.RUnlock()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (b *Builder) GetPartOrderBook(number int) ([]byte, error) {
	//防止切片使用出错
	defer func() {
		if r := recover(); r != nil {
			log.Error("GetPartOrderBook panic : %v", r)
		}
	}()

	////获取读锁
	b.lock.RLock()
	defer b.lock.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"sequence":     b.fullOrderBook.Sequence,
		level3.AskSide: b.fullOrderBook.GetPartOrderBookBySide(level3.AskSide, number),
		level3.BidSide: b.fullOrderBook.GetPartOrderBookBySide(level3.BidSide, number),
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}

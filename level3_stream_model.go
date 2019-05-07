package order_book

import (
	"encoding/json"
)

//Level 3 websocket stream data
type Level3StreamDataModel struct {
	Sequence   string `json:"sequence"`
	Symbol     string `json:"symbol"`
	Type       string `json:"type"`
	Side       string `json:"side"`
	rawMessage json.RawMessage
}

func NewLevel3StreamDataModel(msgData json.RawMessage) (*Level3StreamDataModel, error) {
	l3Data := &Level3StreamDataModel{}

	if err := json.Unmarshal(msgData, l3Data); err != nil {
		return nil, err
	}
	l3Data.rawMessage = msgData

	return l3Data, nil
}

func (l3Data *Level3StreamDataModel) GetRawMessage() json.RawMessage {
	return l3Data.rawMessage
}

const (
	BuySide  = "buy"
	SellSide = "sell"

	LimitOrderType  = "limit"
	MarketOrderType = "market"

	Level3MessageDoneCanceled = "canceled"
	Level3MessageDoneFilled   = "filled"

	Level3MessageReceivedType = "received"
	Level3MessageOpenType     = "open"
	Level3MessageDoneType     = "done"
	Level3MessageMatchType    = "match"
	Level3MessageChangeType   = "change"
)

type Level3StreamDataReceivedModel struct {
	OrderType string `json:"orderType"`
	Side      string `json:"side"`
	//Size      string `json:"size"`
	Price string `json:"price"`
	//Funds     string `json:"funds"`
	OrderId   string `json:"orderId"`
	Time      string `json:"time"`
	ClientOid string `json:"clientOid"`
}

type Level3StreamDataOpenModel struct {
	Side    string `json:"side"`
	Size    string `json:"size"`
	OrderId string `json:"orderId"`
	Price   string `json:"price"`
	Time    string `json:"time"`
}

type Level3StreamDataDoneModel struct {
	Side    string `json:"side"`
	Size    string `json:"size"`
	Reason  string `json:"reason"`
	OrderId string `json:"orderId"`
	Price   string `json:"price"`
	Time    string `json:"time"`
}

type Level3StreamDataMatchModel struct {
	Side         string `json:"side"`
	Size         string `json:"size"`
	Price        string `json:"price"`
	TakerOrderId string `json:"takerOrderId"`
	MakerOrderId string `json:"makerOrderId"`
	Time         string `json:"time"`
	TradeId      string `json:"tradeId"`
}

type Level3StreamDataChangeModel struct {
	Side    string `json:"side"`
	NewSize string `json:"newSize"`
	OldSize string `json:"oldSize"`
	Price   string `json:"price"`
	OrderId string `json:"orderId"`
	Time    string `json:"time"`
}

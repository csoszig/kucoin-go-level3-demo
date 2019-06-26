package builder

import "errors"

type DepthResponse struct {
	Sequence string      `json:"sequence"`
	Asks     [][3]string `json:"asks"` //ask 是 要价，喊价 卖家 卖单 Sort price from low to high
	Bids     [][3]string `json:"bids"` //bid 是 投标，买家 买单 Sort price from high to low
}

type FullOrderBook struct {
	Sequence uint64      `json:"sequence"`
	Asks     [][3]string `json:"asks"` //ask 是 要价，喊价 卖家 卖单 Sort price from low to high
	Bids     [][3]string `json:"bids"` //bid 是 投标，买家 买单 Sort price from high to low
}

func (b *Builder) GetAtomicFullOrderBook() (*DepthResponse, error) {
	rsp, err := b.apiService.AtomicFullOrderBook(b.symbol)
	if err != nil {
		return nil, err
	}

	c := &DepthResponse{}
	if err := rsp.ReadData(c); err != nil {
		return nil, err
	}

	if c.Sequence == "" {
		return nil, errors.New("empty key sequence")
	}

	return c, nil
}

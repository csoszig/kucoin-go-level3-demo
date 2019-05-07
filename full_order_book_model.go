package order_book

type (
	//[3]string{"orderId", "price", "size"}
	Level3OrderBookItem [3]string

	Level3AsksOrderBook [][3]string
	Level3BidsOrderBook [][3]string
)

type FullOrderBookModel struct {
	Sequence string              `json:"sequence"`
	Asks     Level3AsksOrderBook `json:"asks"` //ask Sort price from low to high
	Bids     Level3BidsOrderBook `json:"bids"` //bid Sort price from high to low
}

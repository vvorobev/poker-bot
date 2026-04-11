package fsm

type State string

const (
	StateIdle              State = "idle"
	StateAwaitingPhone     State = "awaiting_phone"
	StateAwaitingBank      State = "awaiting_bank"
	StateAwaitingChipsInput State = "awaiting_chips_input"
	StateAwaitingBuyIn     State = "awaiting_buy_in"
)

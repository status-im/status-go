package stickers

type StickerStatus int

const (
	StatusAvailable StickerStatus = iota
	StatusInstalled
	StatusPending
	StatusPurchased
)

package processor

import (
	"math/rand/v2"
	"time"
)

const bidchars = "0123456789abcdefghjkmnpqrstvwxyz"

type Bid struct{}

func (b Bid) New() string {
	var buf [20]byte
	ms := time.Now().UnixMilli()

	buf[0] = bidchars[(ms>>35)&0x1F]
	buf[1] = bidchars[(ms>>30)&0x1F]
	buf[2] = bidchars[(ms>>25)&0x1F]
	buf[3] = bidchars[(ms>>20)&0x1F]
	buf[4] = bidchars[(ms>>15)&0x1F]
	buf[5] = bidchars[(ms>>10)&0x1F]
	buf[6] = bidchars[(ms>>5)&0x1F]
	buf[7] = bidchars[ms&0x1F]

	r := rand.Uint64()
	buf[8] = bidchars[(r>>55)&0x1F]
	buf[9] = bidchars[(r>>50)&0x1F]
	buf[10] = bidchars[(r>>45)&0x1F]
	buf[11] = bidchars[(r>>40)&0x1F]
	buf[12] = bidchars[(r>>35)&0x1F]
	buf[13] = bidchars[(r>>30)&0x1F]
	buf[14] = bidchars[(r>>25)&0x1F]
	buf[15] = bidchars[(r>>20)&0x1F]
	buf[16] = bidchars[(r>>15)&0x1F]
	buf[17] = bidchars[(r>>10)&0x1F]
	buf[18] = bidchars[(r>>5)&0x1F]
	buf[19] = bidchars[r&0x1F]

	return string(buf[:])
}

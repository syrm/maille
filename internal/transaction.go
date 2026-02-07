package internal

type Transaction struct {
	ID         string
	DatePosted string
	TrnType    string
	TrnAmount  float64
	Name       string
	Account    string
}

type Currency struct {
	ID   uint
	Name string
}

// base58 encodes and decodes int64 values (snowflake IDs) to base58.
//
// The alphabet excludes 0, O, I, l to avoid visual confusion.
// A max int64 (9223372036854775807) encodes to 11 characters.
// Zero allocations on encode (stack buffer). One allocation on decode (string→byte conversion).

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

type Base58 struct {
	decode [128]int8
}

func BuildBase58() Base58 {
	var decode [128]int8

	for i := range decode {
		decode[i] = -1
	}
	for i, c := range alphabet {
		decode[c] = int8(i)
	}

	return Base58{
		decode: decode,
	}
}

// Encode converts a non-negative int64 to a base58 string.
// For negative values the behavior is undefined.
func (b Base58) Encode(id int64) string {
	if id == 0 {
		return "1"
	}
	// 58^11 > math.MaxInt64, so 11 chars is always enough.
	var buf [11]byte
	i := len(buf)
	u := uint64(id)
	for u > 0 {
		i--
		buf[i] = alphabet[u%58]
		u /= 58
	}
	return string(buf[i:])
}

// Decode converts a base58 string back to an int64.
// Returns -1 on invalid input (empty string, invalid character, overflow).
func (b Base58) Decode(s string) int64 {
	if len(s) == 0 || len(s) > 11 {
		return -1
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 128 {
			return -1
		}
		d := b.decode[c]
		if d < 0 {
			return -1
		}
		prev := n
		n = n*58 + uint64(d)
		if n < prev { // overflow
			return -1
		}
	}
	if n > 1<<63-1 { // doesn't fit in int64
		return -1
	}
	return int64(n)
}

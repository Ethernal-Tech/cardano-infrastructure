package sendtx

import "fmt"

type ExchangeRateEntry struct {
	SrcChainID string
	DstChainID string
	Value      float64
}

func NewExchangeRateEntry(srcChainID, dstChainID string, value float64) ExchangeRateEntry {
	return ExchangeRateEntry{
		SrcChainID: srcChainID,
		DstChainID: dstChainID,
		Value:      value,
	}
}

type ExchangeRate map[string]float64

func NewExchangeRate(entries ...ExchangeRateEntry) ExchangeRate {
	r := make(ExchangeRate, len(entries)*2)

	for _, e := range entries {
		r[r.getKey(e.SrcChainID, e.DstChainID)] = e.Value
		r[r.getKey(e.DstChainID, e.SrcChainID)] = 1.0 / e.Value
	}

	return r
}

func (r ExchangeRate) Get(srcChainID, dstChainID string) float64 {
	return setOrDefault(r[r.getKey(srcChainID, dstChainID)], 1)
}

func (r ExchangeRate) getKey(srcChainID, dstChainID string) string {
	return fmt.Sprintf("%s_%s", srcChainID, dstChainID)
}

package deepseek

import "time"

// rates are USD per token.
type rates struct{ hit, miss, out float64 }

// offPeak is DeepSeek's published off-peak rate card, per 1M tokens:
// https://api-docs.deepseek.com/quick_start/pricing (checked 23 Oct 2026).
// Peak rates are exactly double. Change these only from that page.
var offPeak = map[string]rates{
	"deepseek-flash":  {hit: 0.003 / 1e6, miss: 0.15 / 1e6, out: 0.60 / 1e6},
	"deepseek-v4-pro": {hit: 0.022 / 1e6, miss: 0.66 / 1e6, out: 1.98 / 1e6},
}

const peakMultiplier = 2

// mostExpensive prices a model missing from the rate card, so a typo or a new
// model name never makes an answer cheaper than it is.
const mostExpensive = "deepseek-v4-pro"

func offPeakRates(model string) rates {
	if r, ok := offPeak[model]; ok {
		return r
	}
	return offPeak[mostExpensive]
}

func peakRates(model string) rates {
	r := offPeakRates(model)
	return rates{hit: r.hit * peakMultiplier, miss: r.miss * peakMultiplier, out: r.out * peakMultiplier}
}

// IsPeak reports DeepSeek's peak hours: 01:00-04:00 and 06:00-10:00 UTC,
// Monday to Friday. DeepSeek also bills Chinese public holidays off-peak; this
// does not know them and charges peak then, which can cost a customer more
// but never leaves Uplumo short.
func IsPeak(t time.Time) bool {
	t = t.UTC()
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	h := t.Hour()
	return (h >= 1 && h < 4) || (h >= 6 && h < 10)
}

// priceUsage is the USD cost of one response. Cache hit and miss tokens are
// priced separately; a response that reports only prompt_tokens is priced as
// all cache misses.
func priceUsage(model string, u Usage, peak bool) float64 {
	r := offPeakRates(model)
	if peak {
		r = peakRates(model)
	}
	hit, miss := max(u.PromptCacheHitTokens, 0), max(u.PromptCacheMissTokens, 0)
	if hit+miss == 0 {
		miss = max(u.PromptTokens, 0)
	}
	return float64(hit)*r.hit + float64(miss)*r.miss + float64(max(u.CompletionTokens, 0))*r.out
}

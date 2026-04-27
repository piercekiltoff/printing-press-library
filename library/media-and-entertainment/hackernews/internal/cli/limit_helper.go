package cli

import "encoding/json"

// truncateJSONArray returns a JSON array containing at most n elements
// from the input. When the input isn't a JSON array, or n is zero or
// negative, the input is returned unchanged.
//
// Hacker News's Firebase API returns full lists (top 500, best 200,
// etc.) and ignores ?limit query params. The generator's spec-driven
// commands assume the API honors limit; for HN, we have to truncate
// client-side.
func truncateJSONArray(data json.RawMessage, n int) json.RawMessage {
	if n <= 0 {
		return data
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return data
	}
	if len(arr) <= n {
		return data
	}
	out, err := json.Marshal(arr[:n])
	if err != nil {
		return data
	}
	return out
}

// toStringIDs converts a slice of int IDs to strings.
func toStringIDs(ids []int) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = intToStr(id)
	}
	return out
}

func intToStr(n int) string {
	// fmt.Sprintf("%d", n) works fine but a tiny manual path
	// avoids the fmt dependency for a hot loop in fan-out callers.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

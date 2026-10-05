package hardware

func (h Info) Context(trained int, weights, cacheBytes int64) (int, string) {
	target := trained * 3 / 4
	if target <= 0 {
		return 0, "Context metadata unavailable"
	}
	if h.Memory <= 0 || cacheBytes <= 0 || weights <= 0 {
		return min(target, 4096), "Conservative limit: memory requirements are unavailable"
	}
	reserve := max(h.Memory/4, int64(4<<30))
	budget := h.Memory - reserve - weights - (2 << 30)
	if budget <= 0 {
		return min(target, 4096), "Limited memory headroom"
	}
	capacity := budget / cacheBytes / 256 * 256
	if capacity < int64(target) {
		return min(target, max(512, int(capacity))), "Limited by estimated memory headroom"
	}
	return target, "75% of supported context"
}

package library

func (m Metadata) CacheBytesPerToken() int64 {
	number := func(key string) int64 {
		n := m.Numbers[m.Architecture+"."+key]
		if n > 1<<20 {
			return 0
		}
		return int64(n)
	}
	layers, embedding := number("block_count"), number("embedding_length")
	heads, kv := number("attention.head_count"), number("attention.head_count_kv")
	if layers == 0 || heads == 0 || embedding == 0 {
		return 0
	}
	if _, known := m.Numbers[m.Architecture+".attention.head_count_kv"]; !known {
		kv = heads
	}
	if kv == 0 {
		return 0
	}
	key, value := number("attention.key_length"), number("attention.value_length")
	if key == 0 {
		key = embedding / heads
	}
	if value == 0 {
		value = embedding / heads
	}
	if key == 0 || value == 0 {
		return 0
	}
	perLayerHeads := m.NumberArrays[m.Architecture+".attention.head_count_kv"]
	if len(perLayerHeads) == int(layers) {
		pattern := m.NumberArrays[m.Architecture+".attention.sliding_window_pattern"]
		swaKey, swaValue := number("attention.key_length_swa"), number("attention.value_length_swa")
		var total int64
		for layer, count := range perLayerHeads {
			if count == 0 || count > uint64(heads) {
				return 0
			}
			layerKey, layerValue := key, value
			if len(pattern) == len(perLayerHeads) && pattern[layer] != 0 && swaKey > 0 && swaValue > 0 {
				layerKey, layerValue = swaKey, swaValue
			}
			total += 2 * int64(count) * (layerKey + layerValue)
		}
		return total
	}
	return 2 * layers * kv * (key + value)
}

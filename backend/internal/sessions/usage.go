package sessions

func Usage(entries []TranscriptChunk) ChunkUsage {
	latest := map[string]TranscriptChunk{}
	for _, entry := range entries {
		if entry.Usage != nil && (entry.Role == "assistant" || entry.Role == "usage") {
			latest[entry.ID] = entry
		}
	}
	total := ChunkUsage{}
	localOnly := false
	for _, entry := range latest {
		if entry.Role == "assistant" {
			if entry.Provider != "local" {
				localOnly = false
				break
			}
			localOnly = true
		}
	}
	for _, entry := range latest {
		usage := *entry.Usage
		total.CacheUnknown = total.CacheUnknown || (usage.CacheUnknown && entry.Provider != "local" && !(entry.Provider == "" && localOnly))
		total.Input += max(0, usage.Input)
		total.Output += max(0, usage.Output)
		total.Cached += min(max(0, usage.Cached), max(0, usage.Input))
		total.Reasoning += max(0, usage.Reasoning)
	}
	return total
}

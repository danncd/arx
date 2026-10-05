package compaction

import (
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
)

const recentTokens = 15000
const summaryTokens = 2000
const maxImages = 12
const recentImages = 6

func imageCount(messages []provider.Message) int {
	count := 0
	for _, message := range messages {
		count += len(message.Images)
	}
	return count
}

func retentionCut(history []provider.Message, ends []int, through, target int) int {
	cut, used, images := len(history), 0, 0
	for index := len(ends) - 1; index >= 0; index-- {
		start := 0
		if index > 0 {
			start = ends[index-1]
		}
		if start < through {
			break
		}
		size := contextwindow.Messages(history[start:ends[index]])
		count := imageCount(history[start:ends[index]])
		if (used+size > target || images+count > recentImages) && cut < len(history) {
			break
		}
		cut, used = start, used+size
		images += count
	}
	return max(through, cut)
}

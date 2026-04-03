package theme

import "hash/fnv"

func fallbackThemes() []Theme {
	return []Theme{
		{
			Category:   "daily-life",
			Energy:     "gentle",
			Title:      "A Small Part of Today",
			Scenario:   "You are describing one ordinary moment from today that felt more interesting than expected.",
			Warmup:     "What small moment from today would you choose to talk about first?",
			FollowUps:  []string{"Why did that moment stay in your mind?", "What did it say about your mood today?", "Would you want more days like this one?"},
			Vocabulary: []string{"ordinary", "notice", "quietly", "mood"},
			Closing:    "Finish by saying what you want tomorrow to feel like.",
		},
		{
			Category:   "work-and-study",
			Energy:     "stretch",
			Title:      "A Challenge Worth Solving",
			Scenario:   "You are talking about a problem that is difficult but still worth your effort.",
			Warmup:     "What is one challenge you think is worth spending time on right now?",
			FollowUps:  []string{"Why does it matter to you?", "What makes it difficult?", "What would progress look like this month?"},
			Vocabulary: []string{"challenge", "progress", "stuck", "worthwhile"},
			Closing:    "End by naming the next step you would actually take.",
		},
		{
			Category:   "relationships",
			Energy:     "playful",
			Title:      "Someone Easy to Talk To",
			Scenario:   "You are explaining why some people make conversation feel light and easy.",
			Warmup:     "What kind of person is easiest for you to talk to?",
			FollowUps:  []string{"What do they do that helps the conversation?", "What makes the opposite kind of conversation tiring?", "How do you try to be easy to talk to?"},
			Vocabulary: []string{"relaxed", "curious", "awkward", "natural"},
			Closing:    "Finish by saying what makes a conversation memorable for you.",
		},
	}
}

func simpleHash(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}

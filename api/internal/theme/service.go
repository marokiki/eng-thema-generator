package theme

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultModel = "gemini-2.5-flash-lite"

type Prompt struct {
	Category   string   `json:"category"`
	Energy     string   `json:"energy"`
	Title      string   `json:"title"`
	Scenario   string   `json:"scenario"`
	Warmup     string   `json:"warmup"`
	FollowUps  []string `json:"followUps"`
	Vocabulary []string `json:"vocabulary"`
	Closing    string   `json:"closing"`
}

type Theme struct {
	Category   string
	Energy     string
	Title      string
	Scenario   string
	Warmup     string
	FollowUps  []string
	Vocabulary []string
	Closing    string
}

type Service struct {
	apiKey   string
	model    string
	baseURL  string
	client   *http.Client
	fallback []Theme

	mu    sync.RWMutex
	cache map[string]Prompt
}

type categoryTemplate struct {
	title      string
	scenario   string
	warmup     string
	followUps  []string
	vocabulary []string
	closing    string
}

func NewService() *Service {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = defaultModel
	}

	return &Service{
		apiKey:   apiKey,
		model:    model,
		baseURL:  "https://generativelanguage.googleapis.com/v1beta",
		client:   &http.Client{Timeout: 15 * time.Second},
		fallback: fallbackThemes(),
		cache:    make(map[string]Prompt),
	}
}

func (s *Service) Pick(ctx context.Context, category, energy, mode string, now time.Time) Prompt {
	normalizedCategory := normalize(category)
	normalizedEnergy := normalize(energy)
	normalizedMode := normalize(mode)
	if normalizedMode == "" {
		normalizedMode = "random"
	}

	if normalizedMode == "daily" {
		cacheKey := dailyCacheKey(normalizedCategory, normalizedEnergy, now)
		if cached, ok := s.cached(cacheKey); ok {
			return cached
		}

		generated, err := s.generate(ctx, now, normalizedCategory, normalizedEnergy, normalizedMode)
		if err == nil {
			s.store(cacheKey, generated)
			return generated
		}
	}

	if generated, err := s.generate(ctx, now, normalizedCategory, normalizedEnergy, normalizedMode); err == nil {
		return generated
	}

	return s.pickFallback(normalizedCategory, normalizedEnergy, normalizedMode, now)
}

func (s *Service) cached(key string) (Prompt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.cache[key]
	return value, ok
}

func (s *Service) store(key string, prompt Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = prompt
}

func (s *Service) generate(ctx context.Context, now time.Time, category, energy, mode string) (Prompt, error) {
	if s.apiKey == "" {
		return Prompt{}, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	requestBody := geminiRequest{
		SystemInstruction: geminiContent{
			Parts: []geminiPart{{
				Text: "You generate short, easy-to-start English free-talk topics for a learner practicing for about five minutes. Return raw JSON only. Do not wrap the JSON in markdown or code fences.",
			}},
		},
		Contents: []geminiContent{{
			Parts: []geminiPart{{
				Text: s.promptInstruction(now, category, energy, mode),
			}},
		}},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      1.05,
			TopP:             0.95,
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return Prompt{}, err
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", s.baseURL, s.model, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Prompt{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return Prompt{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Prompt{}, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return Prompt{}, fmt.Errorf("gemini request failed: %s", strings.TrimSpace(string(raw)))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Prompt{}, err
	}

	text := parsed.text()
	if text == "" {
		return Prompt{}, fmt.Errorf("gemini returned no text")
	}

	var prompt Prompt
	if err := json.Unmarshal([]byte(cleanJSON(text)), &prompt); err != nil {
		return Prompt{}, err
	}

	return sanitizePrompt(prompt, category, energy), nil
}

func (s *Service) promptInstruction(now time.Time, category, energy, mode string) string {
	return fmt.Sprintf(`Generate one English speaking prompt for a learner practicing free talk for about five minutes.

Constraints:
- The topic must feel natural, personal, and easy to start talking about.
- Avoid politics, trauma, medical issues, explicit content, religion, or anything too heavy.
- Keep the English level around CEFR A2-B2.
- Make it sound fresh and specific, not generic textbook conversation.
- The learner is speaking alone or with a tutor, so the prompt should work without special background knowledge.
- Follow the requested category and energy if provided.
- For mode "daily", make the result feel like a polished prompt of the day, not a random list item.
- Today is %s.

Requested category: %s
Requested energy: %s
Requested mode: %s

Return exactly this JSON shape:
{
  "category": "daily-life | work-and-study | travel-and-places | opinions-and-ideas | relationships | culture-and-media | future-and-goals | food-and-home",
  "energy": "gentle | playful | stretch",
  "title": "short title",
  "scenario": "one or two sentences",
  "warmup": "one easy opening question",
  "followUps": ["question 1", "question 2", "question 3"],
  "vocabulary": ["word1", "word2", "word3", "word4"],
  "closing": "one closing reflection prompt"
}

Rules for the JSON:
- Use exactly 3 follow-up questions.
- Use exactly 4 vocabulary items.
- Keep all values in English.
- Do not add extra keys.`, now.Format("2006-01-02"), displayConstraint(category), displayConstraint(energy), displayConstraint(mode))
}

func (s *Service) pickFallback(category, energy, mode string, now time.Time) Prompt {
	candidates := make([]Theme, 0, len(s.fallback))
	for _, item := range s.fallback {
		categoryMatch := category == "" || category == "any" || normalize(item.Category) == category
		energyMatch := energy == "" || energy == "any" || normalize(item.Energy) == energy
		if categoryMatch && energyMatch {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		return s.buildGenericFallback(category, energy)
	}

	index := int(simpleHash(dailyCacheKey(category, energy, now)) % uint32(len(candidates)))
	if mode == "random" && len(candidates) > 1 {
		index = int(now.UnixNano() % int64(len(candidates)))
	}

	selected := candidates[index]
	return Prompt{
		Category:   selected.Category,
		Energy:     selected.Energy,
		Title:      selected.Title,
		Scenario:   selected.Scenario,
		Warmup:     selected.Warmup,
		FollowUps:  selected.FollowUps,
		Vocabulary: selected.Vocabulary,
		Closing:    selected.Closing,
	}
}

func (s *Service) buildGenericFallback(category, energy string) Prompt {
	resolvedCategory := sanitizeEnum(category, []string{
		"daily-life",
		"work-and-study",
		"travel-and-places",
		"opinions-and-ideas",
		"relationships",
		"culture-and-media",
		"future-and-goals",
		"food-and-home",
	}, category, "daily-life")
	resolvedEnergy := sanitizeEnum(energy, []string{
		"gentle",
		"playful",
		"stretch",
	}, energy, "gentle")

	templates := map[string]categoryTemplate{
		"daily-life": {
			title:      "A Small Part of Your Day",
			scenario:   "You are talking about an ordinary part of life that says something real about you.",
			warmup:     "What small part of your usual day feels more important than people might expect?",
			followUps:  []string{"Why does that part of the day matter to you?", "How does it affect your mood?", "Would you change it if you could?"},
			vocabulary: []string{"routine", "ordinary", "notice", "comfortable"},
			closing:    "Finish by saying what kind of day feels best for you.",
		},
		"work-and-study": {
			title:      "A Skill You Want to Build",
			scenario:   "You are describing a skill or habit you want to improve in work or study.",
			warmup:     "What is one skill you want to get better at this season?",
			followUps:  []string{"Why does it matter now?", "What makes it difficult?", "What kind of practice helps most?"},
			vocabulary: []string{"progress", "practice", "stuck", "improve"},
			closing:    "End by naming one step you could realistically take next.",
		},
		"travel-and-places": {
			title:      "A Place With the Right Mood",
			scenario:   "You are describing a place that feels memorable because of its atmosphere.",
			warmup:     "What place gives you the kind of mood you want more often?",
			followUps:  []string{"What details make that place stand out?", "Who would you take there?", "What time of day fits it best?"},
			vocabulary: []string{"atmosphere", "corner", "familiar", "wander"},
			closing:    "Finish by saying what that place brings out in you.",
		},
		"opinions-and-ideas": {
			title:      "An Opinion You Keep Returning To",
			scenario:   "You are explaining an opinion you have thought about more than once.",
			warmup:     "What is one opinion you have that feels simple at first but becomes more interesting when you explain it?",
			followUps:  []string{"Why do you think people disagree about it?", "Has your view changed over time?", "What makes your view fair rather than extreme?"},
			vocabulary: []string{"opinion", "balanced", "nuance", "convincing"},
			closing:    "End by giving your most honest short version of the opinion.",
		},
		"relationships": {
			title:      "The Kind of Person You Relax Around",
			scenario:   "You are reflecting on the kind of people who make conversation feel easy.",
			warmup:     "What kind of person makes you feel relaxed quickly?",
			followUps:  []string{"What do they do that helps?", "What makes the opposite kind of person difficult to talk to?", "How do you try to make other people comfortable?"},
			vocabulary: []string{"relaxed", "genuine", "awkward", "comfortable"},
			closing:    "Finish by saying what makes a conversation worth remembering.",
		},
		"culture-and-media": {
			title:      "Something You Enjoy Recommending",
			scenario:   "You are talking about a piece of media that is easy and enjoyable to recommend.",
			warmup:     "What movie, show, song, book, or game do you recommend most naturally?",
			followUps:  []string{"What makes it easy to recommend?", "Who might not enjoy it?", "What feeling does it leave people with?"},
			vocabulary: []string{"recommend", "memorable", "taste", "relatable"},
			closing:    "Finish by giving a short reason someone should try it.",
		},
		"future-and-goals": {
			title:      "A Future You Can Picture",
			scenario:   "You are imagining a future version of life that feels realistic enough to work toward.",
			warmup:     "What part of your future is easiest for you to imagine clearly?",
			followUps:  []string{"Why does that future matter to you?", "What would need to change first?", "What would tell you that you are getting closer?"},
			vocabulary: []string{"future", "direction", "realistic", "closer"},
			closing:    "Finish by naming one sign of progress you would like to see.",
		},
		"food-and-home": {
			title:      "Something That Feels Like Home",
			scenario:   "You are describing food, space, or routines that create a feeling of home.",
			warmup:     "What kind of food or home detail makes you feel settled fastest?",
			followUps:  []string{"What memory is connected to it?", "Why does it feel comforting?", "Has your idea of home changed over time?"},
			vocabulary: []string{"settled", "comforting", "familiar", "homemade"},
			closing:    "Finish by saying what home means to you right now.",
		},
	}

	selected := templates[resolvedCategory]
	selected = applyEnergyToTemplate(selected, resolvedEnergy)

	return Prompt{
		Category:   resolvedCategory,
		Energy:     resolvedEnergy,
		Title:      selected.title,
		Scenario:   selected.scenario,
		Warmup:     selected.warmup,
		FollowUps:  selected.followUps,
		Vocabulary: selected.vocabulary,
		Closing:    selected.closing,
	}
}

func applyEnergyToTemplate(template categoryTemplate, energy string) categoryTemplate {
	switch energy {
	case "playful":
		template.title = template.title + " With a Twist"
		template.warmup = strings.Replace(template.warmup, "What", "What fun", 1)
	case "stretch":
		template.title = template.title + " You Need to Explain"
		template.followUps = []string{
			template.followUps[0],
			"What makes this more complicated than it first seems?",
			template.followUps[2],
		}
	}
	return template
}

func sanitizePrompt(prompt Prompt, category, energy string) Prompt {
	prompt.Category = sanitizeEnum(prompt.Category, []string{
		"daily-life",
		"work-and-study",
		"travel-and-places",
		"opinions-and-ideas",
		"relationships",
		"culture-and-media",
		"future-and-goals",
		"food-and-home",
	}, category, "daily-life")
	prompt.Energy = sanitizeEnum(prompt.Energy, []string{
		"gentle",
		"playful",
		"stretch",
	}, energy, "gentle")

	prompt.Title = strings.TrimSpace(prompt.Title)
	prompt.Scenario = strings.TrimSpace(prompt.Scenario)
	prompt.Warmup = strings.TrimSpace(prompt.Warmup)
	prompt.Closing = strings.TrimSpace(prompt.Closing)
	prompt.FollowUps = normalizeList(prompt.FollowUps, 3)
	prompt.Vocabulary = normalizeList(prompt.Vocabulary, 4)

	return prompt
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeList(values []string, expected int) []string {
	out := make([]string, 0, expected)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	for len(out) < expected {
		out = append(out, "Tell me more about that.")
	}
	if len(out) > expected {
		out = out[:expected]
	}
	return out
}

func sanitizeEnum(value string, allowed []string, requested, fallback string) string {
	normalized := normalize(value)
	for _, candidate := range allowed {
		if normalized == candidate {
			return candidate
		}
	}
	if requested != "" && requested != "any" {
		for _, candidate := range allowed {
			if requested == candidate {
				return candidate
			}
		}
	}
	return fallback
}

func cleanJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}

func displayConstraint(value string) string {
	if value == "" {
		return "any"
	}
	return value
}

func dailyCacheKey(category, energy string, now time.Time) string {
	return now.Format("2006-01-02") + ":" + displayConstraint(category) + ":" + displayConstraint(energy)
}

type geminiRequest struct {
	SystemInstruction geminiContent          `json:"systemInstruction"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"topP"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (r geminiResponse) text() string {
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	return r.Candidates[0].Content.Parts[0].Text
}

type Prompt = {
  category: string;
  energy: string;
  title: string;
  scenario: string;
  warmup: string;
  followUps: string[];
  vocabulary: string[];
  closing: string;
};

type ThemeResponse = {
  prompt: Prompt;
  meta: {
    mode: string;
    category: string;
    energy: string;
  };
};

type Advice = {
  summary: string;
  strengths: string[];
  suggestions: string[];
  polished: string;
  focus: string;
};

type AdviceResponse = {
  advice: Advice;
  meta: {
    characters: number;
    words: number;
  };
};

const form = document.querySelector<HTMLFormElement>("#controls");
const copyButton = document.querySelector<HTMLButtonElement>("#copyPrompt");
const coachForm = document.querySelector<HTMLFormElement>("#coachForm");
const coachInput = requiredInput("#coachInput");

const promptBadge = required("#promptBadge");
const promptStatus = required("#promptStatus");
const promptTitle = required("#promptTitle");
const promptScenario = required("#promptScenario");
const promptWarmup = required("#promptWarmup");
const promptFollowups = required("#promptFollowups");
const promptVocabulary = required("#promptVocabulary");
const promptClosing = required("#promptClosing");
const coachStatus = required("#coachStatus");
const coachSummary = required("#coachSummary");
const coachStrengths = required("#coachStrengths");
const coachSuggestions = required("#coachSuggestions");
const coachPolished = required("#coachPolished");
const coachFocus = required("#coachFocus");
const coachCount = required("#coachCount");

form?.addEventListener("submit", (event) => {
  event.preventDefault();
  void loadPrompt();
});

coachForm?.addEventListener("submit", (event) => {
  event.preventDefault();
  void reviewEnglish();
});

coachInput.addEventListener("input", () => {
  renderCoachCount(coachInput.value);
});

copyButton?.addEventListener("click", async () => {
  const text = [
    promptTitle.textContent,
    promptScenario.textContent,
    `Warm-up: ${promptWarmup.textContent}`,
    "Follow-ups:",
    ...Array.from(promptFollowups.querySelectorAll("li")).map((item) => `- ${item.textContent}`),
    `Wrap-up: ${promptClosing.textContent}`,
  ].join("\n");

  try {
    await navigator.clipboard.writeText(text);
    promptStatus.textContent = "Copied to clipboard";
  } catch (_error) {
    promptStatus.textContent = "Clipboard copy is not available in this browser";
  }
});

void loadPrompt();
renderCoachCount(coachInput.value);

async function loadPrompt(): Promise<void> {
  const formData = new FormData(form ?? undefined);
  const params = new URLSearchParams({
    mode: "random",
    category: String(formData.get("category") ?? "any"),
    energy: String(formData.get("energy") ?? "any"),
  });

  promptStatus.textContent = "Generating a natural topic...";
  promptBadge.textContent = "Thinking";

  try {
    const response = await fetch(`/api/theme?${params.toString()}`);
    if (!response.ok) {
      throw new Error(`Request failed with ${response.status}`);
    }

    const data = (await response.json()) as ThemeResponse;
    renderPrompt(data);
  } catch (error) {
    promptBadge.textContent = "Unavailable";
    promptStatus.textContent = "Could not load a prompt";
    promptTitle.textContent = "The theme generator is not responding";
    promptScenario.textContent = error instanceof Error ? error.message : "Unknown error";
    promptWarmup.textContent = "";
    promptFollowups.replaceChildren();
    promptVocabulary.replaceChildren();
    promptClosing.textContent = "";
  }
}

function renderPrompt(data: ThemeResponse): void {
  const { prompt, meta } = data;

  promptBadge.textContent = humanize(prompt.energy);
  promptStatus.textContent = `${humanize(prompt.category)} theme`;
  promptTitle.textContent = prompt.title;
  promptScenario.textContent = prompt.scenario;
  promptWarmup.textContent = prompt.warmup;
  promptClosing.textContent = prompt.closing;

  promptFollowups.replaceChildren(...prompt.followUps.map((item) => createListItem(item)));
  promptVocabulary.replaceChildren(...prompt.vocabulary.map((item) => createListItem(item)));
  coachInput.placeholder = `${prompt.warmup} Because...`;
}

function createListItem(text: string): HTMLLIElement {
  const item = document.createElement("li");
  item.textContent = text;
  return item;
}

async function reviewEnglish(): Promise<void> {
  const text = coachInput.value.trim();

  coachStatus.textContent = "Reviewing your English...";
  if (!text) {
    renderAdvice({
      advice: {
        summary: "Add a short English answer first, then ask for advice.",
        strengths: ["Voice input works best when you say one complete idea."],
        suggestions: ["Say at least one full sentence.", "Add one concrete detail or example.", "End with a clear final sentence."],
        polished: "I want to practice speaking English with one clear idea at a time.",
        focus: "Start with one simple complete sentence.",
      },
      meta: {
        characters: 0,
        words: 0,
      },
    });
    return;
  }

  try {
    const response = await fetch("/api/advice", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ text }),
    });
    if (!response.ok) {
      throw new Error(`Request failed with ${response.status}`);
    }

    const data = (await response.json()) as AdviceResponse;
    renderAdvice(data);
  } catch (error) {
    coachStatus.textContent = "Could not review this text";
    coachSummary.textContent = error instanceof Error ? error.message : "Unknown error";
    coachStrengths.replaceChildren(createListItem("The checker is unavailable right now."));
    coachSuggestions.replaceChildren(createListItem("Try again after the API is available."));
    coachPolished.textContent = text;
    coachFocus.textContent = "Keep the sentence simple and try again.";
  }
}

function renderAdvice(data: AdviceResponse): void {
  const { advice, meta } = data;

  coachStatus.textContent = `${meta.words} words reviewed`;
  coachSummary.textContent = advice.summary;
  coachStrengths.replaceChildren(...advice.strengths.map((item) => createListItem(item)));
  coachSuggestions.replaceChildren(...advice.suggestions.map((item) => createListItem(item)));
  coachPolished.textContent = advice.polished;
  coachFocus.textContent = advice.focus;
}

function renderCoachCount(text: string): void {
  const words = text.trim() ? text.trim().split(/\s+/).length : 0;
  coachCount.textContent = `${words} ${words === 1 ? "word" : "words"}`;
}

function humanize(value: string): string {
  return value
    .split("-")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function required(selector: string): HTMLElement {
  const element = document.querySelector<HTMLElement>(selector);
  if (!element) {
    throw new Error(`Missing element: ${selector}`);
  }
  return element;
}

function requiredInput(selector: string): HTMLTextAreaElement {
  const element = document.querySelector<HTMLTextAreaElement>(selector);
  if (!element) {
    throw new Error(`Missing element: ${selector}`);
  }
  return element;
}

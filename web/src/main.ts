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

const form = document.querySelector<HTMLFormElement>("#controls");
const copyButton = document.querySelector<HTMLButtonElement>("#copyPrompt");

const promptBadge = required("#promptBadge");
const promptStatus = required("#promptStatus");
const promptTitle = required("#promptTitle");
const promptScenario = required("#promptScenario");
const promptWarmup = required("#promptWarmup");
const promptFollowups = required("#promptFollowups");
const promptVocabulary = required("#promptVocabulary");
const promptClosing = required("#promptClosing");

form?.addEventListener("submit", (event) => {
  event.preventDefault();
  void loadPrompt();
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
}

function createListItem(text: string): HTMLLIElement {
  const item = document.createElement("li");
  item.textContent = text;
  return item;
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

# agent-go

[Package documentation](https://pkg.go.dev/github.com/jnb666/agent-go)

A basic set of libraries to call LLM models using the OpenAI chat completions API. Still very incomplete and liable to change.

Why yet another set of libraries?

  - For learning, best way to understand how this stuff works is to build it yourself.
  - To keep it as clean and simple as possible, I'm just adding the functionality which I actually need right now. Existing frameworks seem massively over complicated for what they actually do.

### Setup

Make sure you have at least go version 1.25 installed. Clone the repo and run go mod tidy to pull the dependencies. For playwright-go (which is used by the browser tool) you also need to install the drivers and browsers - see https://github.com/playwright-community/playwright-go

To use the weather tool you'll need to set `OWM_API_KEY` environment variable with the API key obtained from openweathermap.org. You can sign up free for these calls [here](https://home.openweathermap.org/users/sign_up).

To use the Brave search tool you need to set the `BRAVE_API_KEY` environment variable with the API key from [brave.com](https://brave.com/search/api/). They have now dropped new subscriptions to the free plan though. It seems you can sign up for $5 / month with a $5 monthly credit to cover the first 1000 searches .You can see details on that page.

The tests are using `Qwen3.5-9B` model which I am running with [llama.cpp](https://github.com/ggml-org/llama.cpp). There are some notes on getting that setup from [unsloth](https://unsloth.ai/docs/models/qwen3.5). The bigger Qwen3.5 models are even better if you have enough VRAM. You'll need to set `OPENAI_BASE_URL` to point to your server endpoint - e.g. `export OPENAI_BASE_URL=http://localhost:8080/v1`. It should also work with any provider supporting the OpenAI API - just set `OPENAI_BASE_URL` and `OPENAI_API_KEY`.

### Packages

  - [llm](https://github.com/jnb666/agent-go/tree/main/llm) : Provides a simple wrapper around the [openai](https://github.com/openai/openai-go) SDK supporting model selection and chat completions with tool calling and streaming of results. Includes non standard fields to get and send reasoning traces and retrieve llama.cpp timing stats.
  - [agents](https://github.com/jnb666/agent-go/tree/main/agents) : A basic agent implementation supporting setting the system prompt, adding tools and managing the tool call run loop. TODO: add functions to support saving and restoring the message history, pruning the context via sliding window or summarization and prompt templates for dynamic prompt generation.
  - [scrape](https://github.com/jnb666/agent-go/tree/main/scrape) : A web page scraper built using the [playwright-go](https://github.com/playwright-community/playwright-go) and [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown}. Uses a headless Firefox instance.
  - [tools/weather](https://github.com/jnb666/agent-go/tree/main/tools/weather) : The classic weather tool example. Uses the openweathermap.org API.
  - [tools/brave](https://github.com/jnb666/agent-go/tree/main/tools/brave) : Web search tool using the Brave search API. Returns a list of 10 links with title and short description.
  - [tools/browser](https://github.com/jnb666/agent-go/tree/main/tools/browser) : Web page open and find tools using the scrape package. Returns the page text formatted with Markdown.

### Example programs

  - [cmd/chat](https://github.com/jnb666/agent-go/tree/main/cmd/chat) : A simple command line chat app using the llm package.
  - [cmd/tools](https://github.com/jnb666/agent-go/tree/main/cmd/tools) : Extends the chat app to use the agents loop with the weather or browser tools.
  - [cmd/webchat](https://github.com/jnb666/agent-go/tree/main/cmd/webchat) : Web based chat app with Brave search and browser tools.

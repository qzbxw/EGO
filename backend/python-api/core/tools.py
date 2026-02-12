# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import asyncio
import ipaddress
import logging
import os
import re
import socket
import tempfile
import threading
import uuid
from datetime import datetime
from pathlib import Path
from typing import Any, cast
from urllib.parse import urlparse

import docker

# --- Third-party libraries for specific tools
import requests
import sympy
import wikipediaapi
from bs4 import BeautifulSoup

# --- Google GenAI specific imports for tool functionality and error handling
try:
    from google import genai
    from google.genai import errors as genai_errors
except ImportError:
    logging.error(
        "Google GenAI library not found. Please install it using 'pip install google-genai'"
    )
    genai = None  # type: ignore[assignment]
    genai_errors = None  # type: ignore[assignment]

# -----------------------------------------------------------------------------
# --- Local Module Imports
# -----------------------------------------------------------------------------
from .llm_backend import LLMProvider
from .memory_db import VectorMemory
from .prompts import (
    ALTER_EGO_PROMPT_EN,
    EGO_SEARCH_PROMPT_EN,
    SUPEREGO_CRITIC_PROMPT,
    SUPEREGO_OPTIMIZER_PROMPT,
    SUPEREGO_RESEARCHER_PROMPT,
    SUPEREGO_SOLVER_PROMPT,
    SUPEREGO_SYNTHESIZER_PROMPT,
)

# -----------------------------------------------------------------------------
# --- Base Tool Class
# -----------------------------------------------------------------------------


class Tool:
    """
    An abstract base class for all tools that the EGO agent can use.

    This class defines the standard interface that all tools must adhere to,
    ensuring they can be seamlessly integrated into the agent's reasoning loop.
    """

    def __init__(self, name: str, desc: str):
        """
        Initializes a Tool instance.

        Args:
            name (str): The specific name of the tool, which the LLM will use
                        to identify and call it (e.g., "EgoSearch").
            desc (str): A detailed, user-friendly description of what the tool does.
                        This description is crucial as it helps the LLM decide
                        when and how to use the tool.
        """
        self.name = name
        self.desc = desc

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Executes the tool's primary function with a given query.

        This method must be implemented by all subclasses.

        Args:
            query (str): The input query, command, or data for the tool.
            user_id (Optional[str]): The user's unique identifier, which is
                required for tools that operate on user-specific data, such
                as memory. Defaults to None.
            **kwargs: Additional arguments that might be passed by the orchestration layer
                      (e.g., event_callback for streaming).

        Raises:
            NotImplementedError: If a subclass does not implement this method.

        Returns:
            str: The result of the tool's execution, formatted as a string
                 that can be understood by the LLM.
        """
        raise NotImplementedError("The 'use' method must be implemented in all tool subclasses.")


# -----------------------------------------------------------------------------
# --- TOOLS WITH LLM BACKEND DEPENDENCY
# -----------------------------------------------------------------------------
# These tools rely on the LLMProvider to make API calls to Gemini models
# for their functionality.
# -----------------------------------------------------------------------------


class EgoSearch(Tool):
    """A tool that uses the Google Search API via the Gemini model to find real-time information."""

    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="ego_search",
            desc="Performs a Google Search to find real-time information, news, and facts on the web.",
        )
        self.backend = backend

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Executes a Google Search using the Gemini model's built-in tool.

        Args:
            query: The search term or question.
            user_id: Not used by this tool.

        Returns:
            A string containing the search results, or an error message.
        """
        logging.info(f"--- ego_search: Executing with query: '{query}' ---")

        # --- Configure the model to use its native Google Search capability.
        # Using higher temperature (0.7) for better generalization and result diversity
        search_tool = genai.types.Tool(google_search=genai.types.GoogleSearch())
        config = genai.types.GenerateContentConfig(
            temperature=0.7, tools=[search_tool], system_instruction=EGO_SEARCH_PROMPT_EN
        )

        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash", config=config, prompt_parts=[query]
            )
            resp_len = len(response_text) if response_text else 0
            logging.info(f"[ego_search] Response received, length: {resp_len} chars")
            return response_text or "Search returned no results."
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"ego_search failed for query '{query}'. API Error: {e}")
            return "Search is temporarily unavailable due to a technical issue. I will proceed without it or you can ask me to try again later."


class BraveSearch(Tool):
    """A tool that queries Brave Search API for independent web results."""

    def __init__(self):
        super().__init__(
            name="brave_search",
            desc="Searches the web via Brave Search API and returns ranked results with snippets and URLs.",
        )
        self.api_key = os.getenv("BRAVE_SEARCH_API_KEY", "").strip()
        self.result_count = max(1, min(int(os.getenv("BRAVE_SEARCH_RESULT_COUNT", "5")), 10))
        self.timeout = float(os.getenv("BRAVE_SEARCH_TIMEOUT_SECONDS", "15"))
        self.endpoint = os.getenv(
            "BRAVE_SEARCH_ENDPOINT", "https://api.search.brave.com/res/v1/web/search"
        )
        self.public_search_url = os.getenv(
            "BRAVE_PUBLIC_SEARCH_URL", "https://search.brave.com/search"
        )

    @staticmethod
    def _clean_text(text: str) -> str:
        return re.sub(r"\s+", " ", (text or "")).strip()

    @staticmethod
    def _is_valid_result_url(url: str) -> bool:
        if not url:
            return False
        if url.startswith(("javascript:", "mailto:", "tel:", "#")):
            return False
        parsed = urlparse(url)
        if parsed.scheme not in ("http", "https"):
            return False
        host = (parsed.netloc or "").lower()
        if "search.brave.com" in host:
            return False
        return True

    def _search_public_html_sync(self, query: str) -> str:
        headers = {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml",
        }
        params = {"q": query, "source": "web"}
        resp = requests.get(
            self.public_search_url, headers=headers, params=params, timeout=self.timeout
        )
        resp.raise_for_status()

        soup = BeautifulSoup(resp.text, "html.parser")
        results: list[tuple[str, str, str]] = []
        seen_urls: set[str] = set()

        # Preferred selector used by Brave result cards.
        anchors = soup.select("a[data-testid='result-title-a']")

        # Fallback if Brave changes markup.
        if not anchors:
            anchors = soup.select("a[href]")

        for a in anchors:
            href = (a.get("href") or "").strip()
            if not self._is_valid_result_url(href):
                continue
            if href in seen_urls:
                continue

            title = self._clean_text(a.get_text(" ", strip=True))
            if not title:
                continue

            snippet = ""
            parent = a.parent
            for _ in range(4):
                if not parent:
                    break
                snippet_tag = parent.find(
                    ["p", "span", "div"],
                    attrs={"data-testid": re.compile(r"(result-|snippet)", re.I)},
                )
                if snippet_tag:
                    snippet = self._clean_text(snippet_tag.get_text(" ", strip=True))
                    break
                parent = parent.parent

            if not snippet:
                card_text = ""
                p = a.parent
                for _ in range(3):
                    if not p:
                        break
                    card_text = self._clean_text(p.get_text(" ", strip=True))
                    if len(card_text) > len(title):
                        break
                    p = p.parent
                snippet = card_text.replace(title, "", 1).strip(" -:|")

            if len(snippet) > 350:
                snippet = snippet[:350] + "..."

            results.append((title, href, snippet))
            seen_urls.add(href)

            if len(results) >= self.result_count:
                break

        if not results:
            return "No relevant Brave Search results were found."

        lines = ["Brave Search results (public SERP):"]
        for i, (title, url, snippet) in enumerate(results, 1):
            lines.append(f"{i}. {title}")
            lines.append(f"   URL: {url}")
            if snippet:
                lines.append(f"   Snippet: {snippet}")
        return "\n".join(lines)

    def _search_sync(self, query: str) -> str:
        if not self.api_key:
            return self._search_public_html_sync(query)

        headers = {
            "Accept": "application/json",
            "X-Subscription-Token": self.api_key,
        }
        params = {
            "q": query,
            "count": self.result_count,
            "safesearch": "moderate",
        }

        resp = requests.get(self.endpoint, headers=headers, params=params, timeout=self.timeout)
        resp.raise_for_status()
        payload = resp.json()

        web_results = payload.get("web", {}).get("results", []) or []
        news_results = payload.get("news", {}).get("results", []) or []
        merged_results = web_results if web_results else news_results

        if not merged_results:
            return "No relevant Brave Search results were found."

        lines = ["Brave Search results:"]
        for i, item in enumerate(merged_results[: self.result_count], 1):
            title = (item.get("title") or "Untitled").strip()
            url = (item.get("url") or "").strip()
            description = (
                item.get("description")
                or item.get("snippet")
                or item.get("meta_description")
                or ""
            ).strip()
            if len(description) > 350:
                description = description[:350] + "..."
            lines.append(f"{i}. {title}")
            if url:
                lines.append(f"   URL: {url}")
            if description:
                lines.append(f"   Snippet: {description}")
        return "\n".join(lines)

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        logging.info(f"--- brave_search: Executing with query: '{query}' ---")
        try:
            loop = asyncio.get_running_loop()
            return cast("str", await loop.run_in_executor(None, self._search_sync, query))
        except requests.HTTPError as e:
            logging.warning(f"brave_search HTTP error for query '{query}': {e}")
            return f"Brave Search request failed with HTTP error: {e}"
        except requests.RequestException as e:
            logging.warning(f"brave_search request error for query '{query}': {e}")
            return f"Brave Search is temporarily unavailable: {e}"
        except Exception as e:
            logging.error(f"brave_search failed for query '{query}': {e}", exc_info=True)
            return "Brave Search failed unexpectedly."


class WebFetch(Tool):
    """A tool that fetches and extracts clean text from a web page."""

    def __init__(self):
        super().__init__(
            name="web_fetch",
            desc="Fetches a webpage by URL and extracts readable text content for analysis.",
        )
        self.timeout = float(os.getenv("WEB_FETCH_TIMEOUT_SECONDS", "15"))
        self.max_chars = max(2000, int(os.getenv("WEB_FETCH_MAX_CHARS", "12000")))
        self.user_agent = os.getenv(
            "WEB_FETCH_USER_AGENT",
            "EGO-WebFetch/1.0 (+https://example.com; content extraction)",
        )

    def _is_safe_public_url(self, raw_url: str) -> tuple[bool, str]:
        parsed = urlparse(raw_url)
        if parsed.scheme not in ("http", "https"):
            return False, "Only http/https URLs are allowed."

        host = (parsed.hostname or "").strip().lower()
        if not host:
            return False, "URL hostname is missing."
        if host in {"localhost", "127.0.0.1", "::1"}:
            return False, "Localhost URLs are blocked."

        try:
            addr_info = socket.getaddrinfo(host, None)
        except socket.gaierror:
            return False, "Unable to resolve hostname."

        for info in addr_info:
            ip_text = info[4][0]
            ip = ipaddress.ip_address(ip_text)
            if (
                ip.is_private
                or ip.is_loopback
                or ip.is_link_local
                or ip.is_multicast
                or ip.is_reserved
                or ip.is_unspecified
            ):
                return False, "URL resolves to a non-public IP address."

        return True, ""

    def _fetch_sync(self, raw_url: str) -> str:
        url = raw_url.strip()
        is_safe, reason = self._is_safe_public_url(url)
        if not is_safe:
            return f"Blocked URL: {reason}"

        headers = {"User-Agent": self.user_agent}
        resp = requests.get(url, headers=headers, timeout=self.timeout, allow_redirects=True)
        resp.raise_for_status()

        content_type = (resp.headers.get("Content-Type") or "").lower()
        final_url = resp.url

        if "text/html" in content_type:
            soup = BeautifulSoup(resp.text, "html.parser")
            for tag in soup(["script", "style", "noscript", "svg"]):
                tag.decompose()

            title = ""
            if soup.title and soup.title.string:
                title = soup.title.string.strip()

            text = soup.get_text(separator="\n", strip=True)
            text = re.sub(r"\n{3,}", "\n\n", text)
            if len(text) > self.max_chars:
                text = text[: self.max_chars] + "\n...[truncated]"

            lines = [f"Fetched URL: {final_url}"]
            if title:
                lines.append(f"Title: {title}")
            lines.append("Content:")
            lines.append(text or "[No readable text extracted]")
            return "\n".join(lines)

        if content_type.startswith("text/") or "json" in content_type or "xml" in content_type:
            text = resp.text
            if len(text) > self.max_chars:
                text = text[: self.max_chars] + "\n...[truncated]"
            return f"Fetched URL: {final_url}\nContent-Type: {content_type}\n\n{text}"

        return (
            f"Fetched URL: {final_url}\nContent-Type: {content_type}\n"
            "This resource is binary or unsupported for text extraction."
        )

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        logging.info(f"--- web_fetch: Fetching URL from query: '{query}' ---")
        try:
            loop = asyncio.get_running_loop()
            return cast("str", await loop.run_in_executor(None, self._fetch_sync, query))
        except requests.HTTPError as e:
            logging.warning(f"web_fetch HTTP error for '{query}': {e}")
            return f"web_fetch HTTP error: {e}"
        except requests.RequestException as e:
            logging.warning(f"web_fetch request error for '{query}': {e}")
            return f"web_fetch request failed: {e}"
        except Exception as e:
            logging.error(f"web_fetch failed for '{query}': {e}", exc_info=True)
            return f"web_fetch failed: {e}"


class AlterEgo(Tool):
    """A tool that uses a different persona/model to re-analyze a thought or query."""

    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="alter_ego",
            desc="Engages a creative and unconventional persona named 'AlterEgo' to analyze a thought or query from a completely different perspective.",
        )
        self.backend = backend

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Invokes a different model persona to get a creative or alternative viewpoint.

        Args:
            query: The thought or question to be re-analyzed.
            user_id: Not used by this tool.

        Returns:
            A response from the AlterEgo persona.
        """
        logging.info(f"--- alter_ego: Engaging with query: '{query}' ---")

        # --- High temperature for more creative and diverse responses.
        config = genai.types.GenerateContentConfig(
            temperature=0.9, system_instruction=ALTER_EGO_PROMPT_EN
        )

        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash-lite", config=config, prompt_parts=[query]
            )
            logging.info("--- alter_ego: Response successfully generated. ---")
            return response_text
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"alter_ego failed for query '{query}'. API Error: {e}")
            return "AlterEgo is temporarily unavailable. Reverting to my standard mode of thinking."


class EgoCodeExec(Tool):
    """A tool that executes Python code in a secure, isolated Docker container."""

    _build_lock: threading.Lock = threading.Lock()
    _sandbox_dockerfile_fallback = """FROM python:3.11-slim

RUN pip install --no-cache-dir --prefer-binary \\
    numpy \\
    pandas \\
    matplotlib \\
    scipy \\
    requests \\
    beautifulsoup4 \\
    scikit-learn \\
    sympy \\
    pillow \\
    yfinance \\
    openai \\
    anthropic

WORKDIR /sandbox
"""

    def __init__(self, backend: LLMProvider | None = None):
        super().__init__(
            name="ego_code_exec",
            desc="Executes Python code in a secure, isolated Docker environment. Supports libraries like numpy, pandas, matplotlib, requests, and more. Use this for complex data analysis, calculations, and simulations.",
        )
        self.client = docker.from_env()
        self.image_name = os.getenv("EGO_SANDBOX_IMAGE", "ego-sandbox:latest")
        self.volume_name = "sandbox_tmp_data"
        self.sandbox_path = "/app/sandbox_tmp"
        self.auto_build = os.getenv("EGO_CODEEXEC_AUTO_BUILD", "1").lower() in (
            "1",
            "true",
            "yes",
        )
        self.auto_pull = os.getenv("EGO_CODEEXEC_AUTO_PULL", "0").lower() in ("1", "true", "yes")
        self.sandbox_dockerfile_path = Path(
            os.getenv("EGO_SANDBOX_DOCKERFILE_PATH", "/app/Dockerfile.sandbox")
        )

    def _build_sandbox_image_sync(self) -> None:
        """
        Build sandbox image locally if missing.
        Prefers checked-in Dockerfile.sandbox, with an inline fallback.
        """
        with self._build_lock:
            try:
                self.client.images.get(self.image_name)
                return
            except docker.errors.ImageNotFound:
                pass

            logging.info(f"[ego_code_exec] Building sandbox image '{self.image_name}'...")

            if self.sandbox_dockerfile_path.exists():
                build_dir = str(self.sandbox_dockerfile_path.parent)
                dockerfile_name = self.sandbox_dockerfile_path.name
                _, build_logs = self.client.images.build(
                    path=build_dir,
                    dockerfile=dockerfile_name,
                    tag=self.image_name,
                    rm=True,
                    pull=self.auto_pull,
                )
            else:
                logging.warning(
                    f"[ego_code_exec] Sandbox Dockerfile not found at {self.sandbox_dockerfile_path}. "
                    "Using inline fallback Dockerfile."
                )
                with tempfile.TemporaryDirectory(prefix="ego-sandbox-build-") as tmpdir:
                    dockerfile_path = Path(tmpdir) / "Dockerfile"
                    dockerfile_path.write_text(self._sandbox_dockerfile_fallback)
                    _, build_logs = self.client.images.build(
                        path=tmpdir,
                        dockerfile="Dockerfile",
                        tag=self.image_name,
                        rm=True,
                        pull=self.auto_pull,
                    )

            # Emit only concise build status lines.
            for item in build_logs:
                stream_line = item.get("stream")
                if stream_line:
                    line = stream_line.strip()
                    if line:
                        logging.info(f"[ego_code_exec][build] {line}")

            logging.info(f"[ego_code_exec] Sandbox image '{self.image_name}' built successfully.")

    def _ensure_sandbox_image_sync(self) -> None:
        try:
            self.client.images.get(self.image_name)
            return
        except docker.errors.ImageNotFound:
            if not self.auto_build:
                raise
            self._build_sandbox_image_sync()

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Executes Python code in a Docker container.

        Args:
            query: A string containing the Python code to be executed.
            user_id: Not used by this tool.

        Returns:
            The output of the executed code (stdout/stderr) or an error message.
        """
        logging.info("--- ego_code_exec: Executing code in Docker sandbox ---")

        # --- Prepare the script file
        script_id = str(uuid.uuid4())
        filename = f"script_{script_id}.py"
        file_path = Path(self.sandbox_path) / filename

        try:
            # --- Ensure the sandbox directory exists
            file_path.parent.mkdir(parents=True, exist_ok=True)

            # --- Write the code to the shared volume
            file_path.write_text(query)

            logging.info(f"[ego_code_exec] Code written to {file_path}")

            # --- Run the code in the container
            # We use the named volume 'sandbox_tmp_data' which is shared with the host and other containers
            loop = asyncio.get_running_loop()

            def run_container():
                self._ensure_sandbox_image_sync()
                return self.client.containers.run(
                    image=self.image_name,
                    command=f"python /sandbox/{filename}",
                    volumes={self.volume_name: {"bind": "/sandbox", "mode": "rw"}},
                    working_dir="/sandbox",
                    remove=True,
                    stdout=True,
                    stderr=True,
                    mem_limit="256m",
                    nano_cpus=500000000,  # 0.5 CPU
                    network_disabled=False,  # Allow network for requests/yfinance
                )

            # --- Execute the container and capture output
            try:
                result = await loop.run_in_executor(None, run_container)
                output = result.decode("utf-8")
                logging.info("[ego_code_exec] Execution successful")
                return output if output.strip() else "Code executed successfully with no output."
            except docker.errors.ContainerError as e:
                logging.warning(f"[ego_code_exec] Container error: {e}")
                return f"Error during execution:\n{e.stderr.decode('utf-8')}"
            except docker.errors.ImageNotFound:
                logging.error(f"[ego_code_exec] Image {self.image_name} not found after ensure.")
                return f"Error: Sandbox image {self.image_name} is missing and could not be built."
            except docker.errors.BuildError as e:
                logging.error(f"[ego_code_exec] Failed to build sandbox image: {e}", exc_info=True)
                return f"Error: Failed to build sandbox image '{self.image_name}': {e!s}"
            except docker.errors.APIError as e:
                logging.error(
                    f"[ego_code_exec] Docker API error during sandbox ensure/run: {e}",
                    exc_info=True,
                )
                return f"Docker API error during sandbox execution: {e!s}"
            except Exception as e:
                logging.error(f"[ego_code_exec] Unexpected error: {e}", exc_info=True)
                return f"An unexpected error occurred during code execution: {e!s}"

        finally:
            # --- Cleanup the temporary script file
            if file_path.exists():
                try:
                    file_path.unlink()
                except Exception as e:
                    logging.warning(f"[ego_code_exec] Failed to remove temp file {file_path}: {e}")


# -----------------------------------------------------------------------------
# --- TOOLS WITHOUT BACKEND DEPENDENCY
# -----------------------------------------------------------------------------
# These tools operate independently of the LLM provider, using local libraries
# or other services.
# -----------------------------------------------------------------------------


class EgoMemory(Tool):
    """A tool that searches the user's vector memory for relevant past conversations."""

    def __init__(self, vector_memory: VectorMemory):
        super().__init__(
            name="ego_memory",
            desc="Searches a vector memory to find relevant information from the user's past conversations.",
        )
        self.vector_memory = vector_memory

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Performs a semantic search on the user's long-term memory.

        Args:
            query: The search query to find relevant memories.
            user_id: The unique ID of the user whose memory is being searched. This is required.

        Returns:
            A formatted string of relevant memories found, or a message indicating no results.
        """
        logging.info(f"--- ego_memory: Searching for user '{user_id}' with query: '{query}' ---")

        if not user_id:
            logging.warning("ego_memory: A search was attempted without a user_id.")
            return "Error: user_id was not provided, so I cannot access your memory."

        try:
            # --- Perform the search using the provided VectorMemory instance.
            hits = await self.vector_memory.search(user_id, query, top_k=5)
        except Exception as e:
            logging.error(
                f"ego_memory search failed for user '{user_id}'. Error: {e}", exc_info=True
            )
            return "An unexpected error occurred while searching my memory banks."

        if not hits:
            return "No relevant information was found in my memory for your query."

        # --- Format the retrieved memories for the LLM.
        response_lines = ["I found the following relevant snippets in my memory:"]
        for hit in hits:
            try:
                # --- Safely parse and format the timestamp.
                raw_created_at = hit.created_at or ""
                dt = datetime.fromisoformat(raw_created_at.replace("Z", "+00:00"))
                timestamp = f" (from: {dt.strftime('%Y-%m-%d %H:%M')})"
            except (ValueError, TypeError, AttributeError):
                timestamp = ""  # --- Ignore if timestamp is missing or malformed.

            response_lines.append(f'- "{hit.text}" (relevance: {hit.score:.2f}){timestamp}')

        return "\n".join(response_lines)


class EgoCalc(Tool):
    """A tool that performs mathematical calculations using the SymPy library for safety and accuracy."""

    def __init__(self):
        super().__init__(
            name="ego_calc",
            desc="Performs precise mathematical calculations. Accepts expressions like 'sqrt(8) + 5**3'.",
        )

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Safely evaluates a mathematical expression using SymPy.

        Args:
            query: The mathematical expression to evaluate (e.g., "1/3 + pi").
            user_id: Not used by this tool.

        Returns:
            The result of the calculation or a detailed error message.
        """
        logging.info(f"--- ego_calc: Evaluating expression: '{query}' ---")
        try:
            # --- Use sympy.sympify for evaluation.
            # Removing strict=True to be more permissive with whitespace and standard formats.
            expr = sympy.sympify(query)
            result = expr.evalf()
            return f"Result: {result}"
        except Exception:
            # Fallback for very simple arithmetic if SymPy fails for some reason
            try:
                # Basic sanitization: only allow numbers and operators
                if re.match(r"^[0-9\+\-\*\/\(\)\.\s\*\*]+$", query):
                    # Safe enough for basic math fallback
                    res = eval(query, {"__builtins__": None}, {})
                    return f"Result: {res}"
            except Exception:
                pass
            return f"Error: Could not parse or evaluate the expression '{query}'."


class EgoKnowledge(Tool):
    """A tool that uses the Wikipedia API to look up factual information."""

    def __init__(self):
        super().__init__(
            name="ego_knowledge",
            desc="Uses Wikipedia to search for precise information, definitions, and summaries of articles.",
        )
        # --- Initialize the API client with a descriptive user agent.
        self.wiki_wiki = wikipediaapi.Wikipedia(
            user_agent="EGO Agent Knowledge Retrieval (ego-agent/1.0)", language="en"
        )

    def _search_wiki_sync(self, query: str) -> str:
        """
        Synchronous wrapper for the Wikipedia API call to be run in a separate thread.

        Args:
            query: The title of the Wikipedia page to search for.

        Returns:
            The summary of the page, or an error/not-found message.
        """
        try:
            wiki_page = self.wiki_wiki.page(query)
            if wiki_page.exists():
                # --- Return the first section (~1000 chars) for brevity in the agent's context.
                summary = cast("str", wiki_page.summary)
                return summary[:1000] + ("..." if len(summary) > 1000 else "")
            else:
                return f"The page '{query}' was not found on Wikipedia. Please try a different title or check spelling."
        except Exception as e:
            logging.error(
                f"ego_knowledge API call failed for query '{query}'. Error: {e}", exc_info=True
            )
            return "An external error occurred while trying to contact the Wikipedia API."

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Searches for a Wikipedia article in a non-blocking way.

        Args:
            query: The article title to look up.
            user_id: Not used by this tool.

        Returns:
            A summary of the Wikipedia article.
        """
        logging.info(f"--- ego_knowledge: Searching for article: '{query}' ---")
        # --- Run the synchronous network call in a separate thread to avoid blocking
        # --- the main async event loop. This is crucial for async performance.
        loop = asyncio.get_running_loop()
        return cast("str", await loop.run_in_executor(None, self._search_wiki_sync, query))


class ManagePlan(Tool):
    """
    A 'Local Tool' that is intercepted and handled by the Go backend.
    The Python implementation just returns a signal.
    """

    def __init__(self):
        super().__init__(
            name="manage_plan",
            desc="STATE MANAGEMENT. Create and track multi-step plans for complex missions.",
        )

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """Returns a placeholder that will be intercepted by the Go backend."""
        # --- Ensure the query is a valid JSON string even if the model sent a dict
        json_query = query
        logging.info(f"--- ManagePlan: Signal for Go backend: {json_query} ---")
        return f"LOCAL_TOOL_SIGNAL:manage_plan:{json_query}"


class SuperEgo(Tool):
    """
    Multi-agent debate system for complex reasoning.
    Spawns multiple specialized agents that discuss the problem and arrive at a consensus.

    This tool emits special signals that Go backend intercepts for streaming visualization.
    """

    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="super_ego",
            desc="Engages multiple specialized AI agents in structured debate for complex problems requiring diverse expert perspectives. Use this for critical decisions, complex implementations, or when you need adversarial analysis.",
        )
        self.backend = backend

    async def _emit_signal(
        self, signal_type: str, data: dict, event_callback: Any | None = None
    ) -> str:
        """
        Emits a signal that Go backend will intercept and convert to WebSocket event.
        Also streams to event_callback if provided (for Python-native streaming).

        Args:
            signal_type: Type of signal (round_start, agent_start, agent_message, agent_done, round_done)
            data: Signal payload
            event_callback: Optional async callback for real-time streaming

        Returns:
            Signal string in format: SUPEREGO_SIGNAL:{type}:{json_data}
        """
        import json

        signal_data = json.dumps(data, ensure_ascii=False)

        # --- Stream to callback if provided (e.g., from main.py's SSE generator)
        if event_callback:
            try:
                # Map to frontend event type expected by SuperEgoDebate.svelte
                event_type = f"superego_{signal_type}"
                await event_callback({"type": event_type, "data": data})
            except Exception as e:
                logging.warning(f"[super_ego] Failed to invoke event callback: {e}")

        return f"SUPEREGO_SIGNAL:{signal_type}:{signal_data}"

    async def _run_agent(
        self,
        agent_name: str,
        agent_role: str,
        agent_prompt: str,
        query: str,
        debate_history: str,
        round_number: int,
        temperature: float = 0.5,
        event_callback: Any | None = None,
    ) -> tuple[str, list[str]]:
        """
        Runs a single agent with the given prompt and context.

        Args:
            agent_name: Display name of the agent (e.g., "Researcher")
            agent_role: Role identifier (e.g., "researcher", "coder")
            agent_prompt: System instruction for this agent
            query: Original user query
            debate_history: Full conversation history from previous agents
            round_number: Current debate round (1, 2, or 3)
            temperature: LLM temperature for this agent
            event_callback: Optional callback for streaming

        Returns:
            Tuple of (agent's response text, list of signals emitted)
        """
        logging.info(f"[super_ego] Running {agent_name} agent in round {round_number}...")

        signals = []

        # Emit agent start signal
        signals.append(
            await self._emit_signal(
                "agent_start",
                {
                    "agent_name": agent_name,
                    "agent_role": agent_role,
                    "round": round_number,
                },
                event_callback,
            )
        )

        # --- Build the full context for this agent
        full_prompt = f"""
{agent_prompt}

ORIGINAL QUERY:
{query}

DEBATE HISTORY (from previous agents):
{debate_history if debate_history else "[This is the first agent - no prior debate history]"}

YOUR TURN - Provide your analysis:
"""

        config = genai.types.GenerateContentConfig(
            temperature=temperature,
            system_instruction=agent_prompt,
        )

        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash",
                config=config,
                prompt_parts=[full_prompt],
            )
            logging.info(f"[super_ego] {agent_name} completed successfully")

            # Emit agent message signal
            signals.append(
                await self._emit_signal(
                    "agent_message",
                    {
                        "agent_name": agent_name,
                        "agent_role": agent_role,
                        "round": round_number,
                        "message": response_text,
                    },
                    event_callback,
                )
            )

            # Emit agent done signal
            signals.append(
                await self._emit_signal(
                    "agent_done",
                    {
                        "agent_name": agent_name,
                        "agent_role": agent_role,
                        "round": round_number,
                    },
                    event_callback,
                )
            )

            return response_text, signals

        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"[super_ego] {agent_name} failed: {e}")
            error_msg = f"[{agent_name} encountered an error and could not complete the analysis]"

            # Emit error signal
            signals.append(
                await self._emit_signal(
                    "agent_error",
                    {
                        "agent_name": agent_name,
                        "agent_role": agent_role,
                        "round": round_number,
                        "error": str(e),
                    },
                    event_callback,
                )
            )

            return error_msg, signals

    async def use(self, query: str, user_id: str | None = None, **kwargs) -> str:
        """
        Executes the SuperEGO multi-agent debate with signal emission for frontend streaming.

        Args:
            query: The problem or question to analyze
            user_id: Not used by this tool
            **kwargs: Can contain 'event_callback' for streaming

        Returns:
            All signals concatenated with newlines for Go backend to parse and stream
        """
        logging.info(
            f"--- super_ego: Starting multi-agent debate for query: '{query[:100]}...' ---"
        )

        event_callback = kwargs.get("event_callback")

        debate_history = ""
        all_signals = []

        # === ROUND 1: Initial Analysis ===
        all_signals.append(
            await self._emit_signal(
                "round_start", {"round": 1, "title": "Initial Analysis"}, event_callback
            )
        )

        # Researcher
        researcher_output, researcher_signals = await self._run_agent(
            agent_name="Researcher",
            agent_role="researcher",
            agent_prompt=SUPEREGO_RESEARCHER_PROMPT,
            query=query,
            debate_history=debate_history,
            round_number=1,
            temperature=0.3,
            event_callback=event_callback,
        )
        all_signals.extend(researcher_signals)
        debate_history += f"\n\n--- RESEARCHER ---\n{researcher_output}"

        # Solver
        solver_output, solver_signals = await self._run_agent(
            agent_name="Solver",
            agent_role="solver",
            agent_prompt=SUPEREGO_SOLVER_PROMPT,
            query=query,
            debate_history=debate_history,
            round_number=1,
            temperature=0.2,
            event_callback=event_callback,
        )
        all_signals.extend(solver_signals)
        debate_history += f"\n\n--- SOLVER ---\n{solver_output}"

        # Critic
        critic_output, critic_signals = await self._run_agent(
            agent_name="Critic",
            agent_role="critic",
            agent_prompt=SUPEREGO_CRITIC_PROMPT,
            query=query,
            debate_history=debate_history,
            round_number=1,
            temperature=0.7,
            event_callback=event_callback,
        )
        all_signals.extend(critic_signals)
        debate_history += f"\n\n--- CRITIC ---\n{critic_output}"

        all_signals.append(await self._emit_signal("round_done", {"round": 1}, event_callback))

        # === ROUND 2: Refinement ===
        all_signals.append(
            await self._emit_signal(
                "round_start", {"round": 2, "title": "Refinement"}, event_callback
            )
        )

        # Solver (refined)
        solver_refined, solver_refined_signals = await self._run_agent(
            agent_name="Solver",
            agent_role="solver",
            agent_prompt=SUPEREGO_SOLVER_PROMPT
            + "\n\nIMPORTANT: Address the Critic's concerns and refine your solution.",
            query=query,
            debate_history=debate_history,
            round_number=2,
            temperature=0.2,
            event_callback=event_callback,
        )
        all_signals.extend(solver_refined_signals)
        debate_history += f"\n\n--- SOLVER (Refined) ---\n{solver_refined}"

        # Optimizer
        optimizer_output, optimizer_signals = await self._run_agent(
            agent_name="Optimizer",
            agent_role="optimizer",
            agent_prompt=SUPEREGO_OPTIMIZER_PROMPT,
            query=query,
            debate_history=debate_history,
            round_number=2,
            temperature=0.4,
            event_callback=event_callback,
        )
        all_signals.extend(optimizer_signals)
        debate_history += f"\n\n--- OPTIMIZER ---\n{optimizer_output}"

        all_signals.append(await self._emit_signal("round_done", {"round": 2}, event_callback))

        # === ROUND 3: Final Synthesis ===
        all_signals.append(
            await self._emit_signal(
                "round_start", {"round": 3, "title": "Final Synthesis"}, event_callback
            )
        )

        synthesizer_output, synthesizer_signals = await self._run_agent(
            agent_name="Synthesizer",
            agent_role="synthesizer",
            agent_prompt=SUPEREGO_SYNTHESIZER_PROMPT,
            query=query,
            debate_history=debate_history,
            round_number=3,
            temperature=0.5,
            event_callback=event_callback,
        )
        all_signals.extend(synthesizer_signals)

        all_signals.append(await self._emit_signal("round_done", {"round": 3}, event_callback))
        all_signals.append(
            await self._emit_signal(
                "debate_complete", {"summary": synthesizer_output[:200] + "..."}, event_callback
            )
        )

        # Join all signals with newlines - Go backend will parse line by line
        result = "\n".join(all_signals)

        logging.info(
            f"[super_ego] Multi-agent debate completed with {len(all_signals)} signals emitted"
        )
        return result

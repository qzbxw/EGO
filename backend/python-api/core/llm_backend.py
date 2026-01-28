# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import asyncio
import hashlib
import logging
import math
import os
import re
import time
from abc import ABC, abstractmethod
from collections.abc import AsyncGenerator
from contextvars import ContextVar
from pathlib import Path
from typing import Any, ClassVar, cast

from .prompts import CHAT_TITLE_PROMPT_EN

# -----------------------------------------------------------------------------
# --- Provider-Specific SDK Imports
# -----------------------------------------------------------------------------
try:
    import anthropic
    import openai
    from google import genai as google_genai
    from google.genai import errors as genai_errors
except ImportError as e:
    logging.critical(f"Missing a required LLM library. Please install it. Details: {e}")
    # Assign None to allow the program to load, but it will fail on provider instantiation.
    anthropic = openai = google_genai = genai_errors = None  # type: ignore[assignment]

# -----------------------------------------------------------------------------
# --- Safe Error Aliases for google-genai SDK
# -----------------------------------------------------------------------------
# This block safely gets error classes that might exist in different versions
# of the google-generativeai library, preventing crashes on import.
try:
    GENAI_ERR_ResourceExhausted = getattr(genai_errors, "ResourceExhausted", None)
    GENAI_ERR_PermissionDenied = getattr(genai_errors, "PermissionDenied", None)
    GENAI_ERR_ServerError = getattr(genai_errors, "ServerError", None)
    GENAI_ERR_ServiceUnavailable = getattr(genai_errors, "ServiceUnavailable", None)
except Exception:
    # Fallback in case `genai_errors` itself is not available.
    GENAI_ERR_ResourceExhausted = None
    GENAI_ERR_PermissionDenied = None
    GENAI_ERR_ServerError = None
    GENAI_ERR_ServiceUnavailable = None

# -----------------------------------------------------------------------------
# --- Centralized Model Configuration
# -----------------------------------------------------------------------------
# This dictionary is the single source of truth for which models are exposed
# to users for each provider. Editing these lists is the correct way to
# add or remove models from the application's UI.
SUPPORTED_MODELS: dict[str, list[str]] = {
    "openai": [
        "gpt-5",
        "gpt-5-mini",
        "gpt-5-nano",
        "gpt-4.1",
        "gpt-4o",
        "o3",
        "o4-mini",
        "o3-mini",
    ],
    "anthropic": [
        "claude-opus-4-1-20250805",
        "claude-opus-4-20250514",
        "claude-sonnet-4-20250514",
        "claude-3-7-sonnet-latest",
        "claude-3-5-haiku-latest",
    ],
    "grok": [
        "grok-4-latest",
        "grok-3-latest",
    ],
    "gemini": [
        "gemini-2.5-pro",
        "gemini-2.5-flash",
    ],
    # Internal models for the default provider. These are not user-selectable.
    "ego": [
        "gemini-3-flash-preview",
        "gemini-flash-latest",
        "gemini-2.5-pro",
        "gemini-2.5-flash-lite",
    ],
}

# -----------------------------------------------------------------------------
# --- Abstract Base Class
# -----------------------------------------------------------------------------


class LLMProvider(ABC):
    """
    Abstract Base Class for all LLM providers, defining a common interface
    for generating text, streaming responses, and validating API keys.
    """

    def __init__(self, api_key: str | None = None):
        """
        Initializes the provider.

        Args:
            api_key: The API key for the provider. Can be None only for the
                     internal `EgoGeminiProvider`, which manages its own key pool.
        """
        self.api_key = api_key

    @abstractmethod
    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """
        Generates a standard, non-streaming response from the LLM.

        Args:
            preferred_model: The desired model to use for generation.
            config: A provider-specific configuration object (e.g., `genai.types.GenerateContentConfig`).
            prompt_parts: A list of prompt components (text, images, etc.).
            **kwargs: Additional provider-specific arguments.

        Returns:
            A tuple containing:
            - The generated text as a string.
            - An optional dictionary with token usage statistics.
        """
        raise NotImplementedError("Subclasses must implement the 'generate' method.")

    @abstractmethod
    async def embed(
        self, text: str, task_type: str = "RETRIEVAL_DOCUMENT", output_dimensionality: int = 256
    ) -> list[float]:
        """Computes a semantic embedding vector for the given text."""
        raise NotImplementedError("Subclasses must implement the 'embed' method.")

    @abstractmethod
    async def batch_embed(
        self,
        texts: list[str],
        task_type: str = "RETRIEVAL_DOCUMENT",
        output_dimensionality: int = 256,
    ) -> list[list[float]]:
        """Computes semantic embeddings for multiple texts efficiently."""
        raise NotImplementedError("Subclasses must implement the 'batch_embed' method.")

    @abstractmethod
    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """
        Generates a response as an asynchronous stream of text chunks.

        Args:
            model: The model to use for generation.
            prompt: A list of prompt components.
            **kwargs: Additional provider-specific arguments.

        Yields:
            Text chunks of the generated response as strings.
        """
        # This is an async generator; it must be defined with `async def`.
        yield "Not implemented"
        return

    @staticmethod
    @abstractmethod
    def get_supported_models() -> list[str]:
        """
        Returns the curated list of supported models for this provider.
        This should be implemented by each subclass.
        """
        raise NotImplementedError("Subclasses must implement 'get_supported_models'.")

    async def upload_file(self, path: str, mime_type: str) -> Any:
        """
        Uploads a file to the provider (if necessary) and returns a format suitable for prompt_parts.
        For providers that don't support file uploads (like OpenAI/Anthropic in this context),
        it might return None or raise NotImplementedError, or returns text content if applicable.

        Args:
            path: Local path to the file.
            mime_type: MIME type of the file.

        Returns:
            An object that can be added to prompt_parts (e.g. google.genai.types.Part).
        """
        return None

    @staticmethod
    async def list_models(api_key: str) -> list[str]:
        """
        Returns a curated list of models for this provider that the app allows users to select.

        Note: This is intentionally not a live API fetch to avoid exposing all models
        under an account and to keep the UI stable. It returns a pre-defined allowlist.
        """
        return []

    @staticmethod
    @abstractmethod
    async def validate_key(api_key: str) -> bool:
        """
        Performs a lightweight check to validate if an API key is functional.

        Args:
            api_key: The API key to validate.

        Returns:
            True if the key is valid, False otherwise.
        """
        raise NotImplementedError("Subclasses must implement 'validate_key'.")

    @staticmethod
    def _prepare_openai_messages(
        prompt_parts: list[Any], system_instruction: str | None
    ) -> list[dict[str, Any]]:
        """
        A utility to convert the app's internal `prompt_parts` format into the
        standard message list format used by OpenAI, Anthropic, and other compatible APIs.

        It handles raw strings and Google `Part` objects by extracting their text content.
        The function is idempotent; if `prompt_parts` is already a list of messages,
        it returns it unchanged.

        Args:
            prompt_parts: A list of prompt components.
            system_instruction: The system prompt content, if any.

        Returns:
            A list of message dictionaries, e.g.,
            `[{"role": "system", "content": "..."}, {"role": "user", "content": "..."}]`.
        """
        # --- If the input is already in the desired format, do nothing.
        if prompt_parts and isinstance(prompt_parts[0], dict) and "role" in prompt_parts[0]:
            return prompt_parts

        # --- Aggregate all text parts into a single user message.
        user_content = "\n\n".join(
            str(getattr(part, "text", part))
            for part in prompt_parts
            if hasattr(part, "text") or isinstance(part, str)
        )

        messages: list[dict[str, Any]] = []
        if system_instruction:
            messages.append({"role": "system", "content": system_instruction})
        if user_content:
            messages.append({"role": "user", "content": user_content})

        return messages

    @staticmethod
    def _extract_json_prefs(
        config: Any, kwargs: dict[str, Any]
    ) -> tuple[dict[str, Any] | None, bool]:
        """
        Extracts a desired JSON response shape from various config sources.

        This helper checks kwargs, object attributes, and dictionary keys to find
        requests for JSON output, either as a boolean flag or a specific schema.

        Returns:
            A tuple `(json_schema, want_json)` where `json_schema` is a dictionary
            if provided, and `want_json` is True if any form of JSON output is requested.
        """
        schema = kwargs.get("json_schema")
        if schema is None and hasattr(config, "json_schema"):
            schema = config.json_schema
        if schema is None and isinstance(config, dict) and "json_schema" in config:
            schema = config.get("json_schema")

        want_json = bool(kwargs.get("json", False))
        if not want_json and hasattr(config, "json"):
            want_json = bool(config.json)
        if not want_json and isinstance(config, dict) and "json" in config:
            want_json = bool(config.get("json"))

        return (schema, bool(want_json or schema is not None))


# -----------------------------------------------------------------------------
# --- EGO'S INTERNAL GEMINI PROVIDER (DEFAULT)
# -----------------------------------------------------------------------------


class EgoGeminiProvider(LLMProvider):
    """
    Internal provider using a pool of Gemini keys from environment variables.
    This is the default, production-grade provider for the EGO agent, featuring
    automatic API key rotation, cooldowns for rate-limited keys, and model fallbacks.
    """

    # --- Defines fallback models to use if the preferred one fails.
    FALLBACK_CHAINS: ClassVar[dict[str, list[str]]] = {
        "gemini-3-flash-preview": [
            "gemini-flash-latest",
            "gemini-2.5-pro",
            "gemini-2.5-flash-lite",
        ],
        "gemini-flash-latest": ["gemini-2.5-pro", "gemini-2.5-flash-lite"],
        "gemini-2.5-pro": ["gemini-2.5-flash-lite"],
        "gemini-2.5-flash": ["gemini-flash-latest", "gemini-2.5-pro", "gemini-2.5-flash-lite"],
        "gemini-2.5-flash-lite": [],  # Lowest tier fallback
    }

    def __init__(self):
        """
        Initializes the provider, loading keys from the environment and setting up
        the client pool and cooldown management state.
        """
        super().__init__(api_key=None)
        keys_str = os.getenv("GEMINI_BACKEND_API_KEYS")
        if not keys_str:
            raise ValueError("CRITICAL: GEMINI_BACKEND_API_KEYS environment variable is not set.")

        # --- Ensure unique keys and create a client for each.
        self.api_keys = list({k.strip() for k in keys_str.split(",")})
        self.clients_pool = [(key, google_genai.Client(api_key=key)) for key in self.api_keys]

        # --- State for managing key rotation and cooldowns.
        # Now key cooldown is model-specific: (api_key, model_name) -> timestamp
        self._cooldown_until: dict[tuple[str, str], float] = {}
        self._rotator_index: int = 0
        self.pool_size = len(self.clients_pool)

        # --- Cooldown durations configured via environment variables.
        self._quota_cooldown_seconds: float = float(os.getenv("GEMINI_KEY_COOLDOWN_SECONDS", "60"))
        self._transient_cooldown_seconds: float = float(
            os.getenv("GEMINI_TRANSIENT_COOLDOWN_SECONDS", "5")
        )

        logging.info(f"EgoGeminiProvider initialized with {self.pool_size} API key(s).")
        logging.info(f"[KEY ROTATION] Keys loaded: {[f'...{key[-4:]}' for key in self.api_keys]}")

        # --- ContextVars for per-request sticky keys. This is crucial for the Files API,
        # --- as an uploaded file is only accessible via the same API key that uploaded it.
        self._preferred_key_var: ContextVar[str | None] = ContextVar(
            "ego_preferred_key", default=None
        )
        self._last_used_key_var: ContextVar[str | None] = ContextVar(
            "ego_last_used_key", default=None
        )

    def _now(self) -> float:
        """Returns the current monotonic time for consistent cooldown calculations."""
        return time.monotonic()

    def _mark_on_cooldown(self, api_key: str, model_name: str, seconds: float):
        """Sets a cooldown period for a specific API key AND model to prevent repeated failures."""
        self._cooldown_until[(api_key, model_name)] = self._now() + seconds
        logging.warning(
            f"[KEY ROTATION] Key ...{api_key[-4:]} is on cooldown for {model_name} for {seconds:.1f}s."
        )

    def _next_ready_client(
        self, model_name: str
    ) -> tuple[tuple[str, google_genai.Client] | None, float]:
        """
        Finds the next available client in the pool that is not on cooldown for the SPECIFIC model.
        """
        now = self._now()
        earliest_available_time = float("inf")

        # --- Check all keys in a round-robin fashion.
        for i in range(self.pool_size):
            idx = (self._rotator_index + i) % self.pool_size
            api_key, client = self.clients_pool[idx]
            cooldown_end = self._cooldown_until.get((api_key, model_name), 0.0)

            if cooldown_end <= now:
                # --- Found a ready client. Update the rotator for next time.
                self._rotator_index = (idx + 1) % self.pool_size
                logging.info(
                    f"[KEY ROTATION] Using key ...{api_key[-4:]} (index {idx}) for {model_name}"
                )
                return (api_key, client), 0.0

            # --- Keep track of when the next key will be available.
            earliest_available_time = min(earliest_available_time, cooldown_end)

        # --- If no client is ready, calculate the necessary wait time.
        wait_time = (
            max(0, earliest_available_time - now)
            if earliest_available_time != float("inf")
            else 0.5
        )
        logging.warning(
            f"[KEY ROTATION] All keys on cooldown for {model_name}. Next available in {wait_time:.1f}s"
        )
        return None, wait_time

    async def upload_file(self, path: str, mime_type: str) -> Any:
        """
        Uploads a large file (video/audio/large pdf) to Gemini Files API.
        It manages key stickiness: the upload sets a pinned key for subsequent generation calls.
        """
        from google.genai import types

        # 1. Determine which client/key to use.
        # Pick a fresh key that is not on cooldown for 'gemini-2.5-flash' (as a proxy)
        candidate, wait_time = self._next_ready_client("gemini-2.5-flash")
        if not candidate:
            await asyncio.sleep(min(wait_time, 2.0))
            candidate, _ = self._next_ready_client("gemini-2.5-flash")
            if not candidate:
                # Last resort: just use the first client
                candidate = self.clients_pool[0]

        api_key, client = candidate

        # 2. Upload file
        logging.info(
            f"[UPLOAD] Uploading file {Path(path).name} ({mime_type}) using key ...{api_key[-4:]}"
        )
        try:
            # Type ignore because SDK methods are dynamic and confusing MyPy
            file_ref = await client.aio.files.upload(file=path, config={"mime_type": mime_type})

            # 3. Wait for processing if necessary
            start_time = time.time()
            # Type ignore because SDK types are complex and sometimes return None in unexpected ways
            while file_ref.state and file_ref.state.name == "PROCESSING":  # type: ignore[union-attr]
                if time.time() - start_time > 600:  # 10 minute timeout
                    raise RuntimeError(f"File processing timed out for {file_ref.name}")  # type: ignore[union-attr]
                await asyncio.sleep(2)
                file_ref = await client.aio.files.get(name=cast("str", file_ref.name))  # type: ignore[union-attr]

            if file_ref.state and file_ref.state.name == "FAILED":  # type: ignore[union-attr]
                raise RuntimeError(
                    f"File processing failed: {file_ref.error.message}"  # type: ignore[union-attr]
                )

            # 4. Pin this key for the upcoming generate call
            self._preferred_key_var.set(api_key)
            logging.info(f"[UPLOAD] File ready. Key ...{api_key[-4:]} pinned.")

            if not file_ref.uri:  # type: ignore[union-attr]
                raise RuntimeError("File uploaded but no URI returned.")

            return types.Part.from_uri(file_ref.uri, mime_type=mime_type)  # type: ignore[union-attr]

        except Exception as e:
            logging.error(f"[UPLOAD] Failed to upload file: {e}", exc_info=True)
            self._mark_on_cooldown(api_key, "gemini-2.5-flash", self._transient_cooldown_seconds)
            raise e

    async def _execute_with_retries_and_fallbacks(
        self, execution_func, preferred_model: str, **kwargs
    ):
        """
        A robust execution wrapper that handles key rotation, model fallbacks,
        and API errors for any given generation function.
        """
        model_chain = [preferred_model, *self.FALLBACK_CHAINS.get(preferred_model, [])]
        pinned_key: str | None = self._preferred_key_var.get()
        last_exception: Exception | None = None

        for model_name in model_chain:
            kwargs["model_name"] = model_name
            tried_keys: set[str] = set()

            # --- Try all available keys for the current model.
            while len(tried_keys) < self.pool_size:
                candidate, wait_time = (None, 0.0)

                if pinned_key and pinned_key not in tried_keys:
                    for k, c in self.clients_pool:
                        if k == pinned_key:
                            cooldown_end = self._cooldown_until.get((k, model_name), 0.0)
                            if cooldown_end <= self._now():
                                candidate = (k, c)
                            else:
                                wait_time = max(0.0, cooldown_end - self._now())
                            break
                    if not candidate:
                        # If pinned key is busy, we must wait a bit or skip if wait is too long
                        if wait_time > 5.0:
                            tried_keys.add(pinned_key)  # Skip pinned key for this model
                            continue
                        await asyncio.sleep(0.5)
                        continue
                else:
                    candidate, wait_time = self._next_ready_client(model_name)
                    if candidate and candidate[0] in tried_keys:
                        # We already tried all ready keys for this model
                        break

                    if not candidate:
                        # All keys are on cooldown for THIS model.
                        # Don't wait too long, try NEXT model in chain immediately.
                        logging.warning(
                            f"All keys on cooldown for {model_name}. Jumping down the cascade..."
                        )
                        break

                api_key, client = candidate
                tried_keys.add(api_key)
                logging.info(
                    f"[KEY ROTATION] Trying key ...{api_key[-4:]} (attempt {len(tried_keys)}/{self.pool_size}) for {model_name}"
                )

                try:
                    # --- Attempt the actual API call.
                    result = await execution_func(client=client, **kwargs)
                    self._last_used_key_var.set(api_key)
                    return result

                except tuple(
                    e for e in [GENAI_ERR_ResourceExhausted, GENAI_ERR_PermissionDenied] if e
                ) as e:
                    logging.warning(
                        f"Quota exhausted on key ...{api_key[-4:]} for {model_name}. Placing on cooldown."
                    )
                    self._mark_on_cooldown(api_key, model_name, self._quota_cooldown_seconds)
                    last_exception = e

                except genai_errors.ClientError as e:
                    error_code = getattr(e, "code", None) or getattr(e, "status_code", None)
                    if not error_code and hasattr(e, "args") and e.args:
                        error_msg = str(e.args[0]) if e.args else str(e)
                        if "429" in error_msg or "RESOURCE_EXHAUSTED" in error_msg:
                            error_code = 429

                    if error_code == 429 or "RESOURCE_EXHAUSTED" in str(e):
                        logging.warning(
                            f"[KEY ROTATION] Rate limit (429) on key ...{api_key[-4:]} for {model_name}. Continuing..."
                        )
                        self._mark_on_cooldown(api_key, model_name, self._quota_cooldown_seconds)
                    else:
                        logging.warning(
                            f"[KEY ROTATION] Client error on key ...{api_key[-4:]} for {model_name}. Error: {e}"
                        )
                        self._mark_on_cooldown(
                            api_key, model_name, self._transient_cooldown_seconds
                        )
                    last_exception = e

                except tuple(
                    e for e in [GENAI_ERR_ServerError, GENAI_ERR_ServiceUnavailable] if e
                ) as e:
                    logging.warning(
                        f"Transient server error on key ...{api_key[-4:]} for {model_name}. Placing on short cooldown."
                    )
                    self._mark_on_cooldown(api_key, model_name, self._transient_cooldown_seconds)
                    last_exception = e

                except Exception as e:
                    logging.error(
                        f"Unexpected error on key ...{api_key[-4:]} for {model_name}: {e}"
                    )
                    self._mark_on_cooldown(api_key, model_name, self._transient_cooldown_seconds)
                    last_exception = e

            logging.info(f"Finished trying all keys for {model_name}. Moving down the cascade...")

        logging.critical("All API keys and all fallback models in cascade failed.")
        raise last_exception or RuntimeError("Cascade exhausted.")

    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """Implements non-streaming generation with full retry and fallback logic."""

        async def _do_generate(client, model_name, **inner_kwargs):
            return await client.aio.models.generate_content(model=model_name, **inner_kwargs)

        try:
            schema, want_json = self._extract_json_prefs(config, kwargs)

            # --- If config is already a GenerateContentConfig object, use it directly
            if hasattr(config, "__class__") and "GenerateContentConfig" in str(config.__class__):
                # For GenerateContentConfig objects, pass directly but override JSON settings if needed
                tools = getattr(config, "tools", None)
                if tools:
                    logging.info(
                        f"[LLM Backend] Config has {len(tools)} tool(s) - passing through to Gemini"
                    )
                    # IMPORTANT: Gemini API does not support tools + JSON response format
                    # When tools are present, we MUST NOT set response_mime_type to 'application/json'
                    gen_cfg = config
                elif want_json:
                    # Only use JSON settings if NO tools are present
                    from google.genai import types

                    gen_cfg = types.GenerateContentConfig(
                        temperature=getattr(config, "temperature", None),
                        system_instruction=getattr(config, "system_instruction", None),
                        response_mime_type="application/json",
                        response_schema=(
                            schema if schema else getattr(config, "response_schema", None)
                        ),
                    )
                else:
                    gen_cfg = config
            else:
                # --- Normalize dict config for Gemini
                gen_cfg = dict(config) if isinstance(config, dict) else {}
                if hasattr(config, "response_mime_type"):
                    gen_cfg["response_mime_type"] = config.response_mime_type
                if hasattr(config, "response_schema"):
                    gen_cfg["response_schema"] = config.response_schema
                if hasattr(config, "tools"):
                    gen_cfg["tools"] = config.tools
                if hasattr(config, "system_instruction"):
                    gen_cfg["system_instruction"] = config.system_instruction
                if hasattr(config, "temperature"):
                    gen_cfg["temperature"] = config.temperature

                if want_json:
                    gen_cfg["response_mime_type"] = "application/json"
                    if schema:
                        gen_cfg["response_schema"] = schema

            response = await self._execute_with_retries_and_fallbacks(
                _do_generate, preferred_model, contents=prompt_parts, config=gen_cfg
            )
            usage = getattr(response, "usage_metadata", None)
            usage_dict = (
                {
                    "prompt_tokens": getattr(usage, "prompt_token_count", 0),
                    "completion_tokens": getattr(usage, "candidates_token_count", 0),
                    "total_tokens": getattr(usage, "total_token_count", 0),
                }
                if usage
                else None
            )
            return getattr(response, "text", ""), usage_dict
        except Exception as e:
            logging.error(
                f"EgoGemini non-streaming generation failed completely after all retries: {e}",
                exc_info=True,
            )
            # --- Return a user-friendly error to prevent client-side crashes.
            return (
                "Sorry, the service is currently experiencing high load. Please try again shortly.",
                None,
            )

    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """Implements streaming generation with full retry and fallback logic."""

        async def _do_stream(client, model_name, **inner_kwargs):
            return await client.aio.models.generate_content_stream(model=model_name, **inner_kwargs)

        model_chain = [model, *self.FALLBACK_CHAINS.get(model, [])]

        for model_name in model_chain:
            tried_keys: set[str] = set()

            # Try all available keys for the current model
            while len(tried_keys) < self.pool_size:
                candidate, _wait_time = self._next_ready_client(model_name)

                if candidate and candidate[0] in tried_keys:
                    break

                if not candidate:
                    logging.warning(
                        f"[STREAM] All keys on cooldown for {model_name}. Moving down the cascade..."
                    )
                    break

                api_key, client = candidate
                tried_keys.add(api_key)
                logging.info(
                    f"[KEY ROTATION] Streaming with key ...{api_key[-4:]} (attempt {len(tried_keys)}/{self.pool_size}) for {model_name}"
                )

                try:
                    cfg = kwargs.get("config")
                    stream = await _do_stream(
                        client=client, model_name=model_name, contents=prompt, config=cfg
                    )
                    async for chunk in stream:
                        if text := getattr(chunk, "text", None):
                            yield text
                    return

                except genai_errors.ClientError as e:
                    if "429" in str(e) or "RESOURCE_EXHAUSTED" in str(e):
                        self._mark_on_cooldown(api_key, model_name, self._quota_cooldown_seconds)
                    else:
                        self._mark_on_cooldown(
                            api_key, model_name, self._transient_cooldown_seconds
                        )

                except Exception as e:
                    logging.error(
                        f"Unexpected error on streaming key ...{api_key[-4:]} for {model_name}: {e}"
                    )
                    self._mark_on_cooldown(api_key, model_name, self._transient_cooldown_seconds)

            logging.info(f"Cascade: Streaming {model_name} exhausted. Trying next...")

        yield "Service overloaded. Please try again later."

    async def embed(
        self, text: str, task_type: str = "RETRIEVAL_DOCUMENT", output_dimensionality: int = 256
    ) -> list[float]:
        """
        Computes a semantic embedding vector for the given text using Gemini text-embedding-004.

        Args:
            text: The input text to embed.
            task_type: The task type for optimization. Options: RETRIEVAL_DOCUMENT, RETRIEVAL_QUERY,
                      SEMANTIC_SIMILARITY, CLASSIFICATION, CLUSTERING. Default: RETRIEVAL_DOCUMENT.
            output_dimensionality: Output vector dimension (256, 768, 1536, or 3072). Default: 256.

        Returns:
            A list of floats representing the text embedding, normalized to unit length.
        """
        provider = os.getenv("EGO_EMBED_PROVIDER", "gemini").lower()

        # --- Fallback: A local, deterministic embedding algorithm. Only if explicitly set to "local".
        if provider == "local":
            logging.warning(
                "Using deprecated hash-based embeddings. Set EGO_EMBED_PROVIDER=gemini for better results."
            )
            dim = 256
            vec: list[float] = []
            # --- Build per-dimension pseudo-random but deterministic values from hashes.
            for i in range(dim):
                h = hashlib.blake2b((text + "|" + str(i)).encode("utf-8"), digest_size=8).digest()
                iv = int.from_bytes(h, byteorder="little", signed=False)
                v = (iv / 2**64) * 2.0 - 1.0  # Map to [-1, 1]
                vec.append(float(v))
            # --- L2 normalize to unit length.
            norm = math.sqrt(sum(v * v for v in vec)) or 1.0
            return [v / norm for v in vec]

        # --- Default: Use the Gemini embedding API.
        embedding_model = os.getenv("GEMINI_EMBEDDING_MODEL", "text-embedding-004")

        async def _do_embed(client: google_genai.Client, model_name: str, **_: Any) -> list[float]:
            from google.genai import types

            config = types.EmbedContentConfig(
                task_type=task_type, output_dimensionality=output_dimensionality
            )

            resp = await client.aio.models.embed_content(
                model=model_name, contents=text, config=config
            )

            # Extract embedding values
            resp_any = cast("Any", resp)
            if hasattr(resp_any, "embeddings") and resp_any.embeddings:
                embedding_obj = resp_any.embeddings[0]
                if hasattr(embedding_obj, "values") and embedding_obj.values is not None:
                    embedding = list(embedding_obj.values)
                else:
                    embedding = list(embedding_obj)
            elif isinstance(resp_any, dict) and "embedding" in resp_any:
                embedding = resp_any["embedding"]
            else:
                raise RuntimeError(f"Unexpected embedding response format: {type(resp_any)}")

            if not isinstance(embedding, list):
                raise RuntimeError("Embedding response format is invalid.")

            # Normalize to unit length for dimensions < 3072
            if output_dimensionality < 3072:
                import numpy as np

                embedding_np = np.array(embedding, dtype=np.float32)
                norm = np.linalg.norm(embedding_np)
                if norm > 0:
                    embedding = (embedding_np / norm).tolist()

            return [float(v) for v in embedding]

        try:
            return cast(
                "list[float]",
                await self._execute_with_retries_and_fallbacks(_do_embed, embedding_model),
            )
        except Exception as e:
            logging.error(
                f"EgoGemini embed failed completely after all retries: {e}", exc_info=True
            )
            # --- Return a normalized zero-vector fallback.
            return [0.0] * output_dimensionality

    async def batch_embed(
        self,
        texts: list[str],
        task_type: str = "RETRIEVAL_DOCUMENT",
        output_dimensionality: int = 256,
    ) -> list[list[float]]:
        """
        Computes semantic embeddings for multiple texts efficiently using batch processing.

        Args:
            texts: List of input texts to embed.
            task_type: The task type for optimization. Default: RETRIEVAL_DOCUMENT.
            output_dimensionality: Output vector dimension (256, 768, 1536, or 3072). Default: 256.

        Returns:
            A list of embedding vectors, one per input text.
        """
        provider = os.getenv("EGO_EMBED_PROVIDER", "gemini").lower()

        # --- Fallback to sequential embedding for local provider
        if provider == "local":
            logging.warning(
                "Batch embedding not optimized for local provider. Using sequential processing."
            )
            return [await self.embed(text, task_type, output_dimensionality) for text in texts]

        # --- Use Gemini batch embedding API
        embedding_model = os.getenv("GEMINI_EMBEDDING_MODEL", "text-embedding-004")

        async def _do_batch_embed(
            client: google_genai.Client, model_name: str, **_: Any
        ) -> list[list[float]]:
            from google.genai import types

            config = types.EmbedContentConfig(
                task_type=task_type, output_dimensionality=output_dimensionality
            )

            # MyPy struggles with the union type for contents, so we cast to Any
            resp = await client.aio.models.embed_content(
                model=model_name,
                contents=cast("Any", texts),
                config=config,
            )

            # Extract embeddings
            embeddings = []
            resp_any = cast("Any", resp)
            if hasattr(resp_any, "embeddings") and resp_any.embeddings is not None:
                for embedding_obj in resp_any.embeddings:
                    if hasattr(embedding_obj, "values") and embedding_obj.values is not None:
                        embedding = list(cast("Any", embedding_obj.values))
                    else:
                        embedding = list(cast("Any", embedding_obj))

                    # Normalize if needed
                    if output_dimensionality < 3072:
                        import numpy as np

                        embedding_np = np.array(embedding, dtype=np.float32)
                        norm = np.linalg.norm(embedding_np)
                        if norm > 0:
                            embedding = (embedding_np / norm).tolist()

                    embeddings.append([float(v) for v in embedding])
            else:
                raise RuntimeError(f"Unexpected batch embedding response format: {type(resp)}")

            return embeddings

        try:
            return cast(
                "list[list[float]]",
                await self._execute_with_retries_and_fallbacks(_do_batch_embed, embedding_model),
            )
        except Exception as e:
            logging.error(
                f"EgoGemini batch_embed failed: {e}. Falling back to sequential embedding.",
                exc_info=True,
            )
            # Fallback to sequential processing
            return [await self.embed(text, task_type, output_dimensionality) for text in texts]

    @staticmethod
    def get_supported_models() -> list[str]:
        """Returns the internal list of supported EGO (Gemini) models."""
        return SUPPORTED_MODELS["ego"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Not applicable for the internal provider which manages its own key pool."""
        return True

    async def generate_title(self, text: str) -> str:
        """
        Generates a concise chat title (3-6 words) from the given text using Gemini,
        leveraging the provider's retry/fallback logic.

        Args:
            text: The source text to summarize into a short title.

        Returns:
            A short title string. Falls back to "New Chat" on error.
        """

        async def _do_generate(client, model_name, **inner_kwargs):
            return await client.aio.models.generate_content(model=model_name, **inner_kwargs)

        try:
            src = (text or "").strip()
            if not src:
                return "New Chat"

            # Keep it very short and plain text; avoid JSON mode here.
            prompt_parts = [CHAT_TITLE_PROMPT_EN.format(text=src)]
            gen_cfg = {"response_mime_type": "text/plain"}

            response = await self._execute_with_retries_and_fallbacks(
                _do_generate,
                preferred_model="gemini-2.5-flash",
                contents=prompt_parts,
                config=gen_cfg,
            )

            title = (getattr(response, "text", "") or "").strip()
            # Basic cleanup: strip quotes and excessive whitespace/punctuation
            try:
                cleaned = re.sub(
                    r'^["\'\u201c\u201d\u2018\u2019]+|["\'\u201c\u201d\u2018\u2019]+$', "", title
                ).strip()
                cleaned = re.sub(r"\s+", " ", cleaned)
                cleaned = re.sub(r"[\.:;!\s]+$", "", cleaned)
                # Guard against empty result
                return cleaned or "New Chat"
            except Exception:
                return title or "New Chat"
        except Exception as e:
            logging.error(
                f"EgoGemini generate_title failed completely after all retries: {e}", exc_info=True
            )
            return "New Chat"


# -----------------------------------------------------------------------------
# --- External Provider Implementations
# -----------------------------------------------------------------------------
# These classes provide interfaces to external LLM services, assuming the user
# provides their own API key. They implement the abstract methods defined in
# the LLMProvider base class.
# -----------------------------------------------------------------------------


class OpenAIProvider(LLMProvider):
    """
    Provider for OpenAI models like GPT-4o, using the official OpenAI Python SDK.
    """

    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """
        Generates a non-streaming response from an OpenAI model.

        Args:
            preferred_model: The OpenAI model name (e.g., "gpt-4o").
            config: A configuration object, from which `system_instruction` is extracted.
            prompt_parts: A list of prompt components, which will be converted to the
                          OpenAI message format.

        Returns:
            A tuple with the generated text and a dictionary of token usage, or an
            error message and None on failure.
        """
        try:
            client = openai.AsyncOpenAI(api_key=self.api_key)
            # --- We need a special helper for vision models, but for now, the standard one works.
            messages = self._prepare_openai_messages(
                prompt_parts, getattr(config, "system_instruction", None)
            )

            _schema, want_json = self._extract_json_prefs(config, kwargs)
            response_format = None
            if want_json:
                # --- OpenAI supports JSON mode.
                response_format = {"type": "json_object"}

            response = await cast("Any", client.chat.completions).create(
                model=preferred_model,
                messages=cast("Any", messages),
                **({"response_format": response_format} if response_format else {}),
            )
            content = response.choices[0].message.content or ""
            usage = response.usage
            usage_dict = (
                {
                    "prompt_tokens": usage.prompt_tokens,
                    "completion_tokens": usage.completion_tokens,
                    "total_tokens": usage.total_tokens,
                }
                if usage
                else None
            )
            return content, usage_dict
        except Exception as e:
            logging.error(f"OpenAI generate failed: {e}", exc_info=True)
            return f"Error: OpenAI API call failed. Details: {e}", None

    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """
        Generates a streaming response from an OpenAI model.

        Yields:
            Text chunks from the OpenAI API stream.
        """
        try:
            client = openai.AsyncOpenAI(api_key=self.api_key)
            system_instruction = getattr(kwargs.get("config"), "system_instruction", None)
            messages = self._prepare_openai_messages(prompt, system_instruction)

            stream = await client.chat.completions.create(
                model=model,
                messages=cast("Any", messages),
                stream=True,
            )
            async for chunk in cast("Any", stream):
                if content := chunk.choices[0].delta.content:
                    yield content
        except Exception as e:
            logging.error(f"OpenAI stream failed: {e}", exc_info=True)
            yield f"Error: OpenAI API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> list[str]:
        """Returns the curated list of supported OpenAI models."""
        return SUPPORTED_MODELS["openai"]

    @staticmethod
    async def list_models(api_key: str) -> list[str]:
        return SUPPORTED_MODELS["openai"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """
        Validates an OpenAI key by attempting to list available models.
        This is a lightweight and standard way to check key validity.
        """
        try:
            client = openai.AsyncOpenAI(api_key=api_key)
            await client.models.list()
            return True
        except openai.AuthenticationError:
            logging.warning("OpenAI key validation failed: Invalid key provided.")
            return False
        except Exception as e:
            logging.error(f"OpenAI key validation failed with an unexpected error: {e}")
            return False

    async def embed(
        self, text: str, task_type: str = "RETRIEVAL_DOCUMENT", output_dimensionality: int = 256
    ) -> list[float]:
        """Generates embedding using OpenAI's text-embedding-3-small."""
        try:
            client = openai.AsyncOpenAI(api_key=self.api_key)
            resp = await client.embeddings.create(
                input=text, model="text-embedding-3-small", dimensions=output_dimensionality
            )
            return resp.data[0].embedding
        except Exception as e:
            logging.error(f"OpenAI embed failed: {e}")
            return [0.0] * output_dimensionality

    async def batch_embed(
        self,
        texts: list[str],
        task_type: str = "RETRIEVAL_DOCUMENT",
        output_dimensionality: int = 256,
    ) -> list[list[float]]:
        """Generates batch embeddings using OpenAI's text-embedding-3-small."""
        try:
            client = openai.AsyncOpenAI(api_key=self.api_key)
            resp = await client.embeddings.create(
                input=texts, model="text-embedding-3-small", dimensions=output_dimensionality
            )
            return [d.embedding for d in resp.data]
        except Exception as e:
            logging.error(f"OpenAI batch_embed failed: {e}")
            return [[0.0] * output_dimensionality for _ in texts]


class AnthropicProvider(LLMProvider):
    """
    Provider for Anthropic's Claude models, using the official Anthropic Python SDK.
    """

    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """
        Generates a non-streaming response from a Claude model.

        Note: Anthropic has a separate parameter for the system prompt.
        """
        try:
            client = anthropic.AsyncAnthropic(api_key=self.api_key)
            system_instruction = getattr(config, "system_instruction", None)
            # --- Anthropic's API expects the system prompt to be outside the main messages list.
            messages = [
                msg
                for msg in self._prepare_openai_messages(prompt_parts, None)
                if msg["role"] != "system"
            ]

            response = await client.messages.create(
                model=preferred_model,
                messages=cast("Any", messages),
                system=cast("Any", system_instruction),
                max_tokens=4096,  # Anthropic requires max_tokens
            )
            content = "".join(getattr(b, "text", "") for b in response.content)
            usage = response.usage
            usage_dict = (
                {
                    "prompt_tokens": usage.input_tokens,
                    "completion_tokens": usage.output_tokens,
                    "total_tokens": usage.input_tokens + usage.output_tokens,
                }
                if usage
                else None
            )
            return content, usage_dict
        except Exception as e:
            logging.error(f"Anthropic generate failed: {e}", exc_info=True)
            return f"Error: Anthropic API call failed. Details: {e}", None

    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """
        Generates a streaming response from a Claude model.
        """
        try:
            client = anthropic.AsyncAnthropic(api_key=self.api_key)
            cfg = kwargs.get("config")
            system_instruction = getattr(cfg, "system_instruction", None) if cfg else None
            messages = [
                msg
                for msg in self._prepare_openai_messages(prompt, None)
                if msg["role"] != "system"
            ]

            async with client.messages.stream(
                model=model,
                messages=cast("Any", messages),
                system=cast("Any", system_instruction),
                max_tokens=4096,
            ) as stream:
                async for text in stream.text_stream:
                    yield text
        except Exception as e:
            logging.error(f"Anthropic stream failed: {e}", exc_info=True)
            yield f"Error: Anthropic API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> list[str]:
        """Returns the curated list of supported Claude models."""
        return SUPPORTED_MODELS["anthropic"]

    @staticmethod
    async def list_models(api_key: str) -> list[str]:
        return SUPPORTED_MODELS["anthropic"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """
        Validates an Anthropic key by making a minimal, 1-token API call.
        """
        try:
            client = anthropic.AsyncAnthropic(api_key=api_key)
            # --- Use the fastest model available for the check.
            model_to_test = SUPPORTED_MODELS["anthropic"][-1]
            await client.messages.create(
                model=model_to_test, messages=[{"role": "user", "content": "test"}], max_tokens=1
            )
            return True
        except anthropic.AuthenticationError:
            logging.warning("Anthropic key validation failed: Invalid key provided.")
            return False
        except Exception as e:
            logging.error(f"Anthropic key validation failed with an unexpected error: {e}")
            return False

    async def embed(
        self, text: str, task_type: str = "RETRIEVAL_DOCUMENT", output_dimensionality: int = 256
    ) -> list[float]:
        """Anthropic does not have a native embedding API. Return zero vector."""
        return [0.0] * output_dimensionality

    async def batch_embed(
        self,
        texts: list[str],
        task_type: str = "RETRIEVAL_DOCUMENT",
        output_dimensionality: int = 256,
    ) -> list[list[float]]:
        """Anthropic does not have a native embedding API. Return zero vectors."""
        return [[0.0] * output_dimensionality for _ in texts]


class GrokProvider(LLMProvider):
    """
    Provider for xAI's Grok models. It uses an OpenAI-compatible API endpoint,
    so it leverages the OpenAI SDK with a custom base URL.
    """

    BASE_URL = "https://api.x.ai/v1"

    def _client(self) -> openai.AsyncOpenAI:
        """Helper to create a pre-configured OpenAI client pointed at the Grok API."""
        return openai.AsyncOpenAI(api_key=self.api_key, base_url=self.BASE_URL)

    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """Generates a non-streaming response from Grok."""
        try:
            client = self._client()
            messages = self._prepare_openai_messages(
                prompt_parts, getattr(config, "system_instruction", None)
            )

            response = await client.chat.completions.create(
                model=preferred_model,
                messages=cast("Any", messages),
            )
            content = response.choices[0].message.content or ""
            usage = response.usage
            usage_dict = (
                {
                    "prompt_tokens": usage.prompt_tokens,
                    "completion_tokens": usage.completion_tokens,
                    "total_tokens": usage.total_tokens,
                }
                if usage
                else None
            )
            return content, usage_dict
        except Exception as e:
            logging.error(f"Grok generate failed: {e}", exc_info=True)
            return f"Error: Grok API call failed. Details: {e}", None

    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """Generates a streaming response from Grok."""
        try:
            client = self._client()
            cfg = kwargs.get("config")
            system_instruction = getattr(cfg, "system_instruction", None) if cfg else None
            messages = self._prepare_openai_messages(prompt, system_instruction)

            stream = await client.chat.completions.create(
                model=model,
                messages=cast("Any", messages),
                stream=True,
            )
            async for chunk in cast("Any", stream):
                if content := chunk.choices[0].delta.content:
                    yield content
        except Exception as e:
            logging.error(f"Grok stream failed: {e}", exc_info=True)
            yield f"Error: Grok API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> list[str]:
        """Returns the curated list of supported Grok models."""
        return SUPPORTED_MODELS["grok"]

    @staticmethod
    async def list_models(api_key: str) -> list[str]:
        return SUPPORTED_MODELS["grok"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Validates a Grok key by listing models via its OpenAI-compatible endpoint."""
        try:
            client = openai.AsyncOpenAI(api_key=api_key, base_url=GrokProvider.BASE_URL)
            await client.models.list()
            return True
        except openai.AuthenticationError:
            logging.warning("Grok key validation failed: Invalid key.")
            return False
        except Exception as e:
            logging.error(f"Grok key validation failed with unexpected error: {e}")
            return False

    async def embed(
        self, text: str, task_type: str = "RETRIEVAL_DOCUMENT", output_dimensionality: int = 256
    ) -> list[float]:
        """Grok embedding status is uncertain. Return zero vector."""
        return [0.0] * output_dimensionality

    async def batch_embed(
        self,
        texts: list[str],
        task_type: str = "RETRIEVAL_DOCUMENT",
        output_dimensionality: int = 256,
    ) -> list[list[float]]:
        """Grok embedding status is uncertain. Return zero vectors."""
        return [[0.0] * output_dimensionality for _ in texts]


class ExternalGeminiProvider(LLMProvider):
    """
    Provider for external Google Gemini models, for users providing their own key.
    This is simpler than `EgoGeminiProvider` as it doesn't handle key pooling or retries.
    """

    async def upload_file(self, path: str, mime_type: str) -> Any:
        """
        Uploads a large file using the single user-provided key.
        """
        from google.genai import types

        try:
            client = google_genai.Client(api_key=self.api_key)
            logging.info(f"[EXTERNAL UPLOAD] Uploading file {Path(path).name} ({mime_type})")

            file_ref = await client.aio.files.upload(file=path, config={"mime_type": mime_type})

            # Wait for processing
            start_time = time.time()
            while file_ref.state and file_ref.state.name == "PROCESSING":  # type: ignore[union-attr]
                logging.info(
                    f"[EXTERNAL UPLOAD] File {file_ref.name} is processing..."  # type: ignore[union-attr]
                )
                if time.time() - start_time > 600:
                    raise RuntimeError(f"File processing timed out for {file_ref.name}")  # type: ignore[union-attr]
                await asyncio.sleep(2)
                file_ref = await client.aio.files.get(name=cast("str", file_ref.name))  # type: ignore[union-attr]

            if file_ref.state and file_ref.state.name == "FAILED":  # type: ignore[union-attr]
                raise RuntimeError(
                    f"File processing failed: {file_ref.error.message}"  # type: ignore[union-attr]
                )

            logging.info(f"[EXTERNAL UPLOAD] File ready: {file_ref.uri}")  # type: ignore[union-attr]
            if not file_ref.uri:  # type: ignore[union-attr]
                raise RuntimeError("File uploaded but no URI returned.")
            return types.Part.from_uri(file_ref.uri, mime_type=mime_type)  # type: ignore[union-attr]
        except Exception as e:
            logging.error(f"[EXTERNAL UPLOAD] Failed: {e}", exc_info=True)
            raise e

    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: list[Any], **kwargs
    ) -> tuple[str, dict[str, int] | None]:
        """Generates a non-streaming response from Gemini using a single user-provided key."""
        try:
            client = google_genai.Client(api_key=self.api_key)
            # --- Similar logic to EgoGeminiProvider for handling Gemini-specific config.
            schema, want_json = self._extract_json_prefs(config, kwargs)
            gen_cfg = dict(config) if isinstance(config, dict) else {}
            if hasattr(config, "response_mime_type"):
                gen_cfg["response_mime_type"] = config.response_mime_type
            if hasattr(config, "response_schema"):
                gen_cfg["response_schema"] = config.response_schema
            if want_json:
                gen_cfg["response_mime_type"] = "application/json"
                if schema:
                    gen_cfg["response_schema"] = schema

            response = await client.aio.models.generate_content(
                model=preferred_model,
                contents=cast("Any", prompt_parts),
                config=cast("Any", gen_cfg),
            )
            usage = getattr(response, "usage_metadata", None)
            usage_dict = (
                {
                    "prompt_tokens": getattr(usage, "prompt_token_count", 0),
                    "completion_tokens": getattr(usage, "candidates_token_count", 0),
                    "total_tokens": getattr(usage, "total_token_count", 0),
                }
                if usage
                else None
            )
            return getattr(response, "text", ""), usage_dict
        except Exception as e:
            logging.error(f"External Gemini generate failed: {e}", exc_info=True)
            return f"Error: Google Gemini API call failed. Details: {e}", None

    async def generate_synthesis_stream(
        self, model: str, prompt: list[Any], **kwargs
    ) -> AsyncGenerator[str, None]:
        """Generates a streaming response from Gemini using a single user-provided key."""
        try:
            client = google_genai.Client(api_key=self.api_key)
            stream = await client.aio.models.generate_content_stream(
                model=model,
                contents=prompt,
                config=kwargs.get("config"),
            )
            async for chunk in stream:
                if text := getattr(chunk, "text", None):
                    yield text
        except Exception as e:
            logging.error(f"External Gemini stream failed: {e}", exc_info=True)
            yield f"Error: Google Gemini API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> list[str]:
        """Returns the curated list of supported Gemini models for external keys."""
        return SUPPORTED_MODELS["gemini"]

    @staticmethod
    async def list_models(api_key: str) -> list[str]:
        return SUPPORTED_MODELS["gemini"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Validates a Gemini key by performing a minimal generate call."""
        try:
            client = google_genai.Client(api_key=api_key)
            # --- A tiny ping to a fast, permissive model.
            # Use gemini-1.5-flash as it is stable and widely available.
            # Updated to pass 'ping' as a list of contents as per new SDK expectations.
            await client.aio.models.generate_content(
                model="gemini-1.5-flash", contents=cast("Any", ["ping"])
            )
            return True
        except Exception as e:
            logging.warning(f"External Gemini key validation failed: {e}")
            return False


# -----------------------------------------------------------------------------
# --- Factory and Listing Functions
# -----------------------------------------------------------------------------

PROVIDER_MAP: dict[str, type[LLMProvider]] = {
    "openai": OpenAIProvider,
    "anthropic": AnthropicProvider,
    "grok": GrokProvider,
    "gemini": ExternalGeminiProvider,  # type: ignore[type-abstract]
    "ego": EgoGeminiProvider,
}


def get_llm_provider(provider_name: str, api_key: str | None = None) -> LLMProvider:
    """
    Factory function to get an instance of a specific LLM provider.

    This is the primary entry point for creating provider objects throughout the application.

    Args:
        provider_name: The name of the provider (e.g., 'openai', 'ego').
        api_key: The API key for the provider. Required for all external providers.

    Raises:
        ValueError: If the provider name is unknown or if an API key is required
            but not provided.

    Returns:
        An initialized instance of the requested LLMProvider subclass.
    """
    provider_class = PROVIDER_MAP.get(provider_name.lower())
    if not provider_class:
        raise ValueError(f"Unknown LLM provider: '{provider_name}'")

    # --- The internal provider is a special case that doesn't require an API key here.
    if provider_name.lower() == "ego":
        return EgoGeminiProvider()

    if not api_key:
        raise ValueError(f"An API key must be provided for the '{provider_name}' provider.")

    return provider_class(api_key=api_key)


def get_all_supported_models() -> dict[str, list[str]]:
    """
    Returns a dictionary of all externally available providers and their curated model lists.

    This is the definitive function for the frontend to call to learn which models
    are available for users to choose from. It explicitly excludes the internal
    "ego" provider, which is not user-selectable.

    Returns:
        A dictionary where keys are provider names (e.g., "openai") and values
        are lists of their supported model names.
    """
    return {
        provider_name: provider_class.get_supported_models()
        for provider_name, provider_class in PROVIDER_MAP.items()
        if provider_name != "ego"
    }

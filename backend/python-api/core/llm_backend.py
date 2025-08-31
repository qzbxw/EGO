# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import asyncio
from contextvars import ContextVar
import logging
import math
import hashlib
import os
import re
import time
from abc import ABC, abstractmethod
from typing import Any, AsyncGenerator, Dict, List, Optional, Set, Tuple

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
    anthropic = openai = google_genai = genai_errors = None

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
SUPPORTED_MODELS: Dict[str, List[str]] = {
    "openai": [
        "gpt-5", "gpt-5-mini", "gpt-5-nano", "gpt-4.1", "gpt-4o", "o3", "o4-mini", "o3-mini",
    ],
    "anthropic": [
        "claude-opus-4-1-20250805", "claude-opus-4-20250514", "claude-sonnet-4-20250514",
        "claude-3-7-sonnet-latest", "claude-3-5-haiku-latest",
    ],
    "grok": [
        "grok-4-latest", "grok-3-latest",
    ],
    "gemini": [
        "gemini-2.5-pro", "gemini-2.5-flash",
    ],
    # Internal models for the default provider. These are not user-selectable.
    "ego": [
        "gemini-2.5-pro", "gemini-2.5-flash",
    ],
    # OpenRouter model allowlists. These should be edited to control what's exposed.
    "openrouter_free": [
        "openai/gpt-oss-20b:free", "z-ai/glm-4.5-air:free", "tngtech/deepseek-r1t2-chimera:free",
        "deepseek/deepseek-r1-0528:free", "meta-llama/llama-4-maverick:free",
    ],
    # Billing tier: only the paid models you want to expose via OpenRouter.
    "openrouter_billing": [
        "openai/gpt-4.1", "google/gemini-2.5-pro", "anthropic/claude-3.7-sonnet", "openai/gpt-5",
        "x-ai/grok-4", "anthropic/claude-opus-4.1", "openai/o3-pro", "anthropic/claude-opus-4",
        "openai/gpt-4o", "openai/o4-mini-high", "openai/o3", "anthropic/claude-3.7-sonnet:thinking",
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

    def __init__(self, api_key: Optional[str] = None):
        """
        Initializes the provider.

        Args:
            api_key: The API key for the provider. Can be None only for the
                     internal `EgoGeminiProvider`, which manages its own key pool.
        """
        self.api_key = api_key

    @abstractmethod
    async def generate(
        self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs
    ) -> Tuple[str, Optional[Dict[str, int]]]:
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
    async def generate_synthesis_stream(
        self, model: str, prompt: List[Any], **kwargs
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
    def get_supported_models() -> List[str]:
        """
        Returns the curated list of supported models for this provider.
        This should be implemented by each subclass.
        """
        raise NotImplementedError("Subclasses must implement 'get_supported_models'.")

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
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
        prompt_parts: List[Any], system_instruction: Optional[str]
    ) -> List[Dict[str, Any]]:
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
        if prompt_parts and isinstance(prompt_parts[0], dict) and 'role' in prompt_parts[0]:
            return prompt_parts

        # --- Aggregate all text parts into a single user message.
        user_content = "\n\n".join(
            str(getattr(part, 'text', part)) for part in prompt_parts if hasattr(part, 'text') or isinstance(part, str)
        )
        
        messages: List[Dict[str, Any]] = []
        if system_instruction:
            messages.append({"role": "system", "content": system_instruction})
        if user_content:
            messages.append({"role": "user", "content": user_content})
        
        return messages

    @staticmethod
    def _extract_json_prefs(config: Any, kwargs: Dict[str, Any]) -> Tuple[Optional[Dict[str, Any]], bool]:
        """
        Extracts a desired JSON response shape from various config sources.

        This helper checks kwargs, object attributes, and dictionary keys to find
        requests for JSON output, either as a boolean flag or a specific schema.

        Returns:
            A tuple `(json_schema, want_json)` where `json_schema` is a dictionary
            if provided, and `want_json` is True if any form of JSON output is requested.
        """
        schema = kwargs.get('json_schema')
        if schema is None and hasattr(config, 'json_schema'):
            schema = getattr(config, 'json_schema')
        if schema is None and isinstance(config, dict) and 'json_schema' in config:
            schema = config.get('json_schema')

        want_json = bool(kwargs.get('json', False))
        if not want_json and hasattr(config, 'json'):
            want_json = bool(getattr(config, 'json'))
        if not want_json and isinstance(config, dict) and 'json' in config:
            want_json = bool(config.get('json'))

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
    FALLBACK_CHAINS: Dict[str, List[str]] = {
        'gemini-2.5-pro': ['gemini-2.5-flash'],
        'gemini-2.5-flash': [],  # Flash has no further fallbacks.
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
        self.api_keys = list(set(k.strip() for k in keys_str.split(",")))
        self.clients_pool = [(key, google_genai.Client(api_key=key)) for key in self.api_keys]
        
        # --- State for managing key rotation and cooldowns.
        self._cooldown_until: Dict[str, float] = {key: 0.0 for key in self.api_keys}
        self._rotator_index: int = 0
        self.pool_size = len(self.clients_pool)
        
        # --- Cooldown durations configured via environment variables.
        self._quota_cooldown_seconds: float = float(os.getenv("GEMINI_KEY_COOLDOWN_SECONDS", "3600"))  # 1 hour for quota exhaustion
        self._transient_cooldown_seconds: float = float(os.getenv("GEMINI_TRANSIENT_COOLDOWN_SECONDS", "10"))
        
        logging.info(f"EgoGeminiProvider initialized with {self.pool_size} API key(s).")
        logging.info(f"[KEY ROTATION] Keys loaded: {[f'...{key[-4:]}' for key in self.api_keys]}")

        # --- ContextVars for per-request sticky keys. This is crucial for the Files API,
        # --- as an uploaded file is only accessible via the same API key that uploaded it.
        self._preferred_key_var: ContextVar[str | None] = ContextVar("ego_preferred_key", default=None)
        self._last_used_key_var: ContextVar[str | None] = ContextVar("ego_last_used_key", default=None)

    def _now(self) -> float:
        """Returns the current monotonic time for consistent cooldown calculations."""
        return time.monotonic()

    def _mark_on_cooldown(self, api_key: str, seconds: float):
        """Sets a cooldown period for a specific API key to prevent repeated failures."""
        self._cooldown_until[api_key] = self._now() + seconds
        logging.warning(f"[KEY ROTATION] Key ...{api_key[-4:]} is on cooldown for {seconds:.1f}s. Total keys on cooldown: {sum(1 for end_time in self._cooldown_until.values() if end_time > self._now())}/{self.pool_size}")

    def _next_ready_client(self) -> Tuple[Optional[Tuple[str, google_genai.Client]], float]:
        """
        Finds the next available client in the pool that is not on cooldown.

        It implements a round-robin rotation to distribute load.

        Returns:
            A tuple containing:
            - A `(api_key, client)` tuple if a ready client is found, else `None`.
            - The minimum time in seconds to wait before a key becomes available
              if all are on cooldown.
        """
        now = self._now()
        earliest_available_time = float('inf')
        
        logging.debug(f"[KEY ROTATION] Checking {self.pool_size} keys, current rotator index: {self._rotator_index}")
        
        # --- Check all keys in a round-robin fashion.
        for i in range(self.pool_size):
            idx = (self._rotator_index + i) % self.pool_size
            api_key, client = self.clients_pool[idx]
            cooldown_end = self._cooldown_until.get(api_key, 0.0)
            
            logging.debug(f"[KEY ROTATION] Key ...{api_key[-4:]} cooldown_end: {cooldown_end}, now: {now}, ready: {cooldown_end <= now}")
            
            if cooldown_end <= now:
                # --- Found a ready client. Update the rotator for next time.
                self._rotator_index = (idx + 1) % self.pool_size
                logging.info(f"[KEY ROTATION] Using key ...{api_key[-4:]} (index {idx}), next rotator index: {self._rotator_index}")
                return (api_key, client), 0.0
            
            # --- Keep track of when the next key will be available.
            earliest_available_time = min(earliest_available_time, cooldown_end)

        # --- If no client is ready, calculate the necessary wait time.
        wait_time = max(0, earliest_available_time - now) if earliest_available_time != float('inf') else 0.5
        logging.warning(f"[KEY ROTATION] All {self.pool_size} keys are on cooldown. Next available in {wait_time:.1f}s")
        return None, wait_time

    async def _execute_with_retries_and_fallbacks(self, execution_func, preferred_model: str, **kwargs):
        """
        A robust execution wrapper that handles key rotation, model fallbacks,
        and API errors for any given generation function.

        This is the core of the provider's resilience. It iterates through the
        model fallback chain (e.g., pro -> flash). For each model, it iterates
        through all available API keys. If an API call fails with a recoverable
        error (like rate limiting), the key is put on cooldown, and the next key is tried.

        Args:
            execution_func: The async function to execute (e.g., `_do_generate`).
            preferred_model: The initial model to try.
            **kwargs: Arguments to pass to `execution_func`.

        Raises:
            Exception: Re-raises the last captured exception if all keys and all
                fallback models fail, ensuring critical failures are not silenced.

        Returns:
            The result of the `execution_func` on success.
        """
        model_chain = [preferred_model] + self.FALLBACK_CHAINS.get(preferred_model, [])
        # --- Honor a "pinned" key for the current request if present (e.g., for Files API).
        pinned_key: Optional[str] = self._preferred_key_var.get()
        last_exception: Optional[Exception] = None

        for model_name in model_chain:
            kwargs['model_name'] = model_name
            tried_keys: Set[str] = set()

            # --- Try all available keys for the current model.
            while len(tried_keys) < self.pool_size:
                candidate, wait_time = (None, 0.0)
                # --- If a key is pinned, we must use it.
                if pinned_key and pinned_key not in tried_keys:
                    # Find the client for the pinned key if it's not on cooldown.
                    for k, c in self.clients_pool:
                        if k == pinned_key:
                            cooldown_end = self._cooldown_until.get(k, 0.0)
                            if cooldown_end <= self._now():
                                candidate = (k, c)
                            else:
                                wait_time = max(0.0, cooldown_end - self._now())
                            break
                    if not candidate:
                        logging.debug(f"Pinned key ...{pinned_key[-4:]} is on cooldown. Waiting {wait_time:.1f}s...")
                        await asyncio.sleep(wait_time or 0.5)
                        continue
                else:
                    # --- Get the next available client from the rotating pool, but skip already tried keys
                    for attempt in range(self.pool_size):
                        candidate, wait_time = self._next_ready_client()
                        if candidate and candidate[0] not in tried_keys:
                            break
                        elif candidate:
                            # This key was already tried, force rotation to next
                            logging.debug(f"Key ...{candidate[0][-4:]} already tried, rotating to next")
                            continue
                    
                    if not candidate:
                        logging.debug(f"All keys are on cooldown. Waiting {wait_time:.1f}s...")
                        await asyncio.sleep(wait_time)
                        continue

                api_key, client = candidate
                tried_keys.add(api_key)
                logging.info(f"[KEY ROTATION] Trying key ...{api_key[-4:]} (attempt {len(tried_keys)}/{self.pool_size})")
                
                try:
                    # --- Attempt the actual API call.
                    result = await execution_func(client=client, **kwargs)
                    # --- Success! Record the key used and pin it for this request.
                    self._last_used_key_var.set(api_key)
                    return result
                
                # --- Handle specific, recoverable API errors by putting the key on cooldown.
                except tuple(e for e in [GENAI_ERR_ResourceExhausted, GENAI_ERR_PermissionDenied] if e) as e:
                    logging.warning(f"Quota exhausted on key ...{api_key[-4:]} for {model_name}. Placing on long cooldown.")
                    self._mark_on_cooldown(api_key, self._quota_cooldown_seconds)
                    last_exception = e
                
                except genai_errors.ClientError as e:
                    # Extract status code from the error message or response
                    error_code = getattr(e, 'code', None) or getattr(e, 'status_code', None)
                    if not error_code and hasattr(e, 'args') and e.args:
                        # Try to extract from error message
                        error_msg = str(e.args[0]) if e.args else str(e)
                        if '429' in error_msg or 'RESOURCE_EXHAUSTED' in error_msg:
                            error_code = 429
                    
                    if error_code == 429 or 'RESOURCE_EXHAUSTED' in str(e):  # Too Many Requests
                        logging.warning(f"[KEY ROTATION] Rate limit (429) on key ...{api_key[-4:]} for {model_name}. Placing on long cooldown. Continuing to next key...")
                        self._mark_on_cooldown(api_key, self._quota_cooldown_seconds)
                    else:
                        logging.warning(f"[KEY ROTATION] Client error on key ...{api_key[-4:]} for {model_name}. Placing on short cooldown. Error: {e}")
                        self._mark_on_cooldown(api_key, self._transient_cooldown_seconds)
                    last_exception = e
                    # Continue to next key instead of breaking
                
                except tuple(e for e in [GENAI_ERR_ServerError, GENAI_ERR_ServiceUnavailable] if e) as e:
                    logging.warning(f"Transient server error on key ...{api_key[-4:]} for {model_name}. Placing on short cooldown.")
                    self._mark_on_cooldown(api_key, self._transient_cooldown_seconds)
                    last_exception = e

                except Exception as e:
                    logging.error(f"Unexpected error on key ...{api_key[-4:]} for {model_name}: {e}", exc_info=True)
                    self._mark_on_cooldown(api_key, self._transient_cooldown_seconds)
                    last_exception = e
            
            logging.info(f"All keys failed for model {model_name}. Trying next fallback model if available.")
        
        # --- If we exit all loops, every key and every model has failed.
        logging.critical("All API keys and fallback models failed to generate a response.")
        raise last_exception or RuntimeError("All API keys and fallback models failed.")

    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
        """Implements non-streaming generation with full retry and fallback logic."""
        async def _do_generate(client, model_name, **inner_kwargs):
            return await client.aio.models.generate_content(model=model_name, **inner_kwargs)

        try:
            schema, want_json = self._extract_json_prefs(config, kwargs)
            # --- Normalize config for Gemini.
            gen_cfg = dict(config) if isinstance(config, dict) else {}
            if hasattr(config, 'response_mime_type'): gen_cfg['response_mime_type'] = config.response_mime_type
            if hasattr(config, 'response_schema'): gen_cfg['response_schema'] = config.response_schema
            
            if want_json:
                gen_cfg['response_mime_type'] = 'application/json'
                if schema: gen_cfg['response_schema'] = schema

            response = await self._execute_with_retries_and_fallbacks(
                _do_generate, preferred_model, contents=prompt_parts, config=gen_cfg
            )
            usage = getattr(response, 'usage_metadata', None)
            usage_dict = {
                "prompt_tokens": getattr(usage, 'prompt_token_count', 0),
                "completion_tokens": getattr(usage, 'candidates_token_count', 0),
                "total_tokens": getattr(usage, 'total_token_count', 0),
            } if usage else None
            return getattr(response, 'text', ''), usage_dict
        except Exception as e:
            logging.error(f"EgoGemini non-streaming generation failed completely after all retries: {e}", exc_info=True)
            # --- Return a user-friendly error to prevent client-side crashes.
            return ("Sorry, the service is currently experiencing high load. Please try again shortly.", None)

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """Implements streaming generation with full retry and fallback logic."""
        async def _do_stream(client, model_name, **inner_kwargs):
            return await client.aio.models.generate_content_stream(model=model_name, **inner_kwargs)
        
        model_chain = [model] + self.FALLBACK_CHAINS.get(model, [])
        last_exception: Optional[Exception] = None
        
        for model_name in model_chain:
            tried_keys: Set[str] = set()
            
            # Try all available keys for the current model
            while len(tried_keys) < self.pool_size:
                # Get the next available client from the rotating pool, but skip already tried keys
                candidate = None
                for attempt in range(self.pool_size):
                    candidate, wait_time = self._next_ready_client()
                    if candidate and candidate[0] not in tried_keys:
                        break
                    elif candidate:
                        # This key was already tried, force rotation to next
                        logging.debug(f"Key ...{candidate[0][-4:]} already tried, rotating to next")
                        continue
                
                if not candidate:
                    logging.debug(f"All keys are on cooldown. Waiting {wait_time:.1f}s...")
                    await asyncio.sleep(wait_time)
                    continue

                api_key, client = candidate
                tried_keys.add(api_key)
                logging.info(f"[KEY ROTATION] Streaming with key ...{api_key[-4:]} (attempt {len(tried_keys)}/{self.pool_size})")
                
                try:
                    cfg = kwargs.get('config')
                    stream = await _do_stream(client=client, model_name=model_name, contents=prompt, config=cfg)
                    # Success! Stream the response
                    async for chunk in stream:
                        if text := getattr(chunk, 'text', None):
                            yield text
                    return  # Successfully streamed, exit
                    
                except genai_errors.ClientError as e:
                    error_code = getattr(e, 'code', None) or getattr(e, 'status_code', None)
                    if not error_code and hasattr(e, 'args') and e.args:
                        error_msg = str(e.args[0]) if e.args else str(e)
                        if '429' in error_msg or 'RESOURCE_EXHAUSTED' in error_msg:
                            error_code = 429
                    
                    if error_code == 429 or 'RESOURCE_EXHAUSTED' in str(e):
                        logging.warning(f"[KEY ROTATION] Rate limit (429) on streaming key ...{api_key[-4:]} for {model_name}. Placing on long cooldown. Continuing to next key...")
                        self._mark_on_cooldown(api_key, self._quota_cooldown_seconds)
                    else:
                        logging.warning(f"[KEY ROTATION] Client error on streaming key ...{api_key[-4:]} for {model_name}. Placing on short cooldown. Error: {e}")
                        self._mark_on_cooldown(api_key, self._transient_cooldown_seconds)
                    last_exception = e
                    # Continue to next key
                    
                except Exception as e:
                    logging.error(f"Unexpected error on streaming key ...{api_key[-4:]} for {model_name}: {e}", exc_info=True)
                    self._mark_on_cooldown(api_key, self._transient_cooldown_seconds)
                    last_exception = e
            
            logging.info(f"All keys failed for streaming model {model_name}. Trying next fallback model if available.")
        
        # If we exit all loops, every key and every model has failed
        logging.error(f"EgoGemini streaming generation failed completely after all retries: {last_exception}", exc_info=True)
        yield "Sorry, the service is currently overloaded. Please try again."

    async def embed(self, text: str) -> List[float]:
        """
        Computes a semantic embedding vector for the given text.

        It uses a local, deterministic method by default for speed and cost-effectiveness.
        It can be configured to use the Gemini embedding API via an environment variable.

        Args:
            text: The input text to embed.

        Returns:
            A list of floats representing the text embedding.
        """
        provider = os.getenv("EGO_EMBED_PROVIDER", "local").lower()
        
        # --- Default: A local, deterministic embedding algorithm. No API calls needed.
        if provider != "gemini":
            dim = 256
            vec: List[float] = []
            # --- Build per-dimension pseudo-random but deterministic values from hashes.
            for i in range(dim):
                h = hashlib.blake2b((text + "|" + str(i)).encode("utf-8"), digest_size=8).digest()
                iv = int.from_bytes(h, byteorder="little", signed=False)
                v = (iv / 2**64) * 2.0 - 1.0 # Map to [-1, 1]
                vec.append(float(v))
            # --- L2 normalize to unit length.
            norm = math.sqrt(sum(v*v for v in vec)) or 1.0
            return [v / norm for v in vec]
        
        # --- Optional: Use the Gemini embedding API.
        EMBEDDING_MODEL = os.getenv("GEMINI_EMBEDDING_MODEL", "text-embedding-004")

        async def _do_embed(client: google_genai.Client, model_name: str, **_: Any) -> List[float]:
            resp = await client.aio.models.embed_content(model=model_name, content=text)
            embedding = resp.get("embedding")
            if not isinstance(embedding, list):
                raise RuntimeError("Embedding response format is invalid.")
            return [float(v) for v in embedding]

        try:
            return await self._execute_with_retries_and_fallbacks(_do_embed, EMBEDDING_MODEL)
        except Exception as e:
            logging.error(f"EgoGemini embed failed completely after all retries: {e}", exc_info=True)
            # --- Return a zero-vector fallback. VectorMemory can handle this.
            return [0.0] * 768 # Match Gemini embedding dimension

    @staticmethod
    def get_supported_models() -> List[str]:
        """Returns the internal list of supported EGO (Gemini) models."""
        return SUPPORTED_MODELS["ego"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Not applicable for the internal provider which manages its own key pool."""
        return True

    async def generate_title(self, text: str) -> str:
        """
        Generates a concise chat title (3–6 words) from the given text using Gemini,
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
            prompt_parts = [
                (
                    "Create a very short, clear chat title (3-6 words) for the following user input. "
                    "Do not wrap in quotes. Avoid trailing punctuation. Use natural title case.\n\n"
                    f"Text:\n{src}"
                )
            ]
            gen_cfg = {"response_mime_type": "text/plain"}

            response = await self._execute_with_retries_and_fallbacks(
                _do_generate,
                preferred_model="gemini-2.5-flash-lite",
                contents=prompt_parts,
                config=gen_cfg,
            )

            title = (getattr(response, "text", "") or "").strip()
            # Basic cleanup: strip quotes and excessive whitespace/punctuation
            try:
                cleaned = re.sub(r'^[\"\'\“\”\‘\’]+|[\"\'\“\”\‘\’]+$', "", title).strip()
                cleaned = re.sub(r"\s+", " ", cleaned)
                cleaned = re.sub(r"[\.:;!\s]+$", "", cleaned)
                # Guard against empty result
                return cleaned or "New Chat"
            except Exception:
                return title or "New Chat"
        except Exception as e:
            logging.error(f"EgoGemini generate_title failed completely after all retries: {e}", exc_info=True)
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
    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
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
            messages = self._prepare_openai_messages(prompt_parts, getattr(config, 'system_instruction', None))
            
            schema, want_json = self._extract_json_prefs(config, kwargs)
            response_format = None
            if want_json:
                # --- OpenAI supports JSON mode.
                response_format = {"type": "json_object"}
            
            response = await client.chat.completions.create(
                model=preferred_model,
                messages=messages,
                **({"response_format": response_format} if response_format else {})
            )
            content = response.choices[0].message.content or ""
            usage = response.usage
            usage_dict = {
                "prompt_tokens": usage.prompt_tokens,
                "completion_tokens": usage.completion_tokens,
                "total_tokens": usage.total_tokens
            } if usage else None
            return content, usage_dict
        except Exception as e:
            logging.error(f"OpenAI generate failed: {e}", exc_info=True)
            return f"Error: OpenAI API call failed. Details: {e}", None

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """
        Generates a streaming response from an OpenAI model.

        Yields:
            Text chunks from the OpenAI API stream.
        """
        try:
            client = openai.AsyncOpenAI(api_key=self.api_key)
            cfg = kwargs.get('config')
            messages = self._prepare_openai_messages(prompt, cfg.system_instruction if cfg else None)
            
            stream = await client.chat.completions.create(
                model=model,
                messages=messages,
                stream=True,
            )
            async for chunk in stream:
                if content := chunk.choices[0].delta.content:
                    yield content
        except Exception as e:
            logging.error(f"OpenAI stream failed: {e}", exc_info=True)
            yield f"Error: OpenAI API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> List[str]:
        """Returns the curated list of supported OpenAI models."""
        return SUPPORTED_MODELS["openai"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
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

class AnthropicProvider(LLMProvider):
    """
    Provider for Anthropic's Claude models, using the official Anthropic Python SDK.
    """
    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
        """
        Generates a non-streaming response from a Claude model.

        Note: Anthropic has a separate parameter for the system prompt.
        """
        try:
            client = anthropic.AsyncAnthropic(api_key=self.api_key)
            system_instruction = getattr(config, 'system_instruction', None)
            # --- Anthropic's API expects the system prompt to be outside the main messages list.
            messages = [msg for msg in self._prepare_openai_messages(prompt_parts, None) if msg["role"] != "system"]
            
            response = await client.messages.create(
                model=preferred_model,
                messages=messages,
                system=system_instruction,
                max_tokens=4096 # Anthropic requires max_tokens
            )
            content = "".join(getattr(b, 'text', '') for b in response.content)
            usage = response.usage
            usage_dict = {
                "prompt_tokens": usage.input_tokens,
                "completion_tokens": usage.output_tokens,
                "total_tokens": usage.input_tokens + usage.output_tokens
            } if usage else None
            return content, usage_dict
        except Exception as e:
            logging.error(f"Anthropic generate failed: {e}", exc_info=True)
            return f"Error: Anthropic API call failed. Details: {e}", None

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """
        Generates a streaming response from a Claude model.
        """
        try:
            client = anthropic.AsyncAnthropic(api_key=self.api_key)
            system_instruction = kwargs.get('config').system_instruction
            messages = [msg for msg in self._prepare_openai_messages(prompt, None) if msg["role"] != "system"]
            
            async with client.messages.stream(
                model=model,
                messages=messages,
                system=system_instruction,
                max_tokens=4096
            ) as stream:
                async for text in stream.text_stream:
                    yield text
        except Exception as e:
            logging.error(f"Anthropic stream failed: {e}", exc_info=True)
            yield f"Error: Anthropic API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> List[str]:
        """Returns the curated list of supported Claude models."""
        return SUPPORTED_MODELS["anthropic"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
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
                model=model_to_test,
                messages=[{"role": "user", "content": "test"}],
                max_tokens=1
            )
            return True
        except anthropic.AuthenticationError:
            logging.warning("Anthropic key validation failed: Invalid key provided.")
            return False
        except Exception as e:
            logging.error(f"Anthropic key validation failed with an unexpected error: {e}")
            return False

class GrokProvider(LLMProvider):
    """
    Provider for xAI's Grok models. It uses an OpenAI-compatible API endpoint,
    so it leverages the OpenAI SDK with a custom base URL.
    """
    BASE_URL = "https://api.x.ai/v1"

    def _client(self) -> openai.AsyncOpenAI:
        """Helper to create a pre-configured OpenAI client pointed at the Grok API."""
        return openai.AsyncOpenAI(api_key=self.api_key, base_url=self.BASE_URL)

    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
        """Generates a non-streaming response from Grok."""
        try:
            client = self._client()
            messages = self._prepare_openai_messages(prompt_parts, getattr(config, 'system_instruction', None))
            
            response = await client.chat.completions.create(
                model=preferred_model,
                messages=messages
            )
            content = response.choices[0].message.content or ""
            usage = response.usage
            usage_dict = {
                "prompt_tokens": usage.prompt_tokens,
                "completion_tokens": usage.completion_tokens,
                "total_tokens": usage.total_tokens
            } if usage else None
            return content, usage_dict
        except Exception as e:
            logging.error(f"Grok generate failed: {e}", exc_info=True)
            return f"Error: Grok API call failed. Details: {e}", None

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """Generates a streaming response from Grok."""
        try:
            client = self._client()
            cfg = kwargs.get('config')
            messages = self._prepare_openai_messages(prompt, cfg.system_instruction if cfg else None)
            
            stream = await client.chat.completions.create(
                model=model,
                messages=messages,
                stream=True,
            )
            async for chunk in stream:
                if content := chunk.choices[0].delta.content:
                    yield content
        except Exception as e:
            logging.error(f"Grok stream failed: {e}", exc_info=True)
            yield f"Error: Grok API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> List[str]:
        """Returns the curated list of supported Grok models."""
        return SUPPORTED_MODELS["grok"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
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

class ExternalGeminiProvider(LLMProvider):
    """
    Provider for external Google Gemini models, for users providing their own key.
    This is simpler than `EgoGeminiProvider` as it doesn't handle key pooling or retries.
    """
    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
        """Generates a non-streaming response from Gemini using a single user-provided key."""
        try:
            client = google_genai.Client(api_key=self.api_key)
            # --- Similar logic to EgoGeminiProvider for handling Gemini-specific config.
            schema, want_json = self._extract_json_prefs(config, kwargs)
            gen_cfg = dict(config) if isinstance(config, dict) else {}
            if hasattr(config, 'response_mime_type'): gen_cfg['response_mime_type'] = config.response_mime_type
            if hasattr(config, 'response_schema'): gen_cfg['response_schema'] = config.response_schema
            if want_json:
                gen_cfg['response_mime_type'] = 'application/json'
                if schema: gen_cfg['response_schema'] = schema

            response = await client.aio.models.generate_content(
                model=preferred_model, contents=prompt_parts, generation_config=gen_cfg
            )
            usage = getattr(response, 'usage_metadata', None)
            usage_dict = {
                "prompt_tokens": getattr(usage, 'prompt_token_count', 0),
                "completion_tokens": getattr(usage, 'candidates_token_count', 0),
                "total_tokens": getattr(usage, 'total_token_count', 0),
            } if usage else None
            return getattr(response, 'text', ''), usage_dict
        except Exception as e:
            logging.error(f"External Gemini generate failed: {e}", exc_info=True)
            return f"Error: Google Gemini API call failed. Details: {e}", None

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """Generates a streaming response from Gemini using a single user-provided key."""
        try:
            client = google_genai.Client(api_key=self.api_key)
            stream = await client.aio.models.generate_content_stream(
                model=model, contents=prompt, generation_config=kwargs.get('config')
            )
            async for chunk in stream:
                if text := getattr(chunk, 'text', None):
                    yield text
        except Exception as e:
            logging.error(f"External Gemini stream failed: {e}", exc_info=True)
            yield f"Error: Google Gemini API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> List[str]:
        """Returns the curated list of supported Gemini models for external keys."""
        return SUPPORTED_MODELS["gemini"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
        return SUPPORTED_MODELS["gemini"]

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Validates a Gemini key by performing a minimal generate call."""
        try:
            client = google_genai.Client(api_key=api_key)
            # --- A tiny ping to a fast, permissive model.
            await client.aio.models.generate_content(model="gemini-1.5-flash-latest", contents=["ping"]) 
            return True
        except Exception as e:
            logging.warning(f"External Gemini key validation failed: {e}")
            return False

# -----------------------------------------------------------------------------
# --- OpenRouter Providers (via OpenAI-compatible SDK)
# -----------------------------------------------------------------------------

class _OpenRouterBaseProvider(LLMProvider):
    """
    An abstract base provider for OpenRouter.

    It uses the OpenAI SDK with a custom `base_url` to communicate with the
    OpenRouter API. Subclasses must define their own `ALLOWED_MODELS` list.
    """
    BASE_URL = "https://openrouter.ai/api/v1"
    ALLOWED_MODELS: List[str] = [] # Subclasses must override this.

    def _client(self) -> openai.AsyncOpenAI:
        """Helper to create a pre-configured OpenAI client for OpenRouter."""
        return openai.AsyncOpenAI(api_key=self.api_key, base_url=self.BASE_URL)

    @staticmethod
    def _prepare_openai_vision_messages(prompt_parts: List[Any], system_instruction: Optional[str]) -> List[Dict[str, Any]]:
        """
        Builds OpenAI-compatible messages that support multimodal inputs (text + images)
        for use with OpenRouter. This is a more advanced version of the base helper.

        It can process various input formats for images, including URLs and base64 data.

        Args:
            prompt_parts: A list of prompt components (strings, dicts with image data, etc.).
            system_instruction: The system prompt content.

        Returns:
            A list of messages in the format expected by OpenAI's vision models.
        """
        # --- If already in messages format, return as-is.
        if prompt_parts and isinstance(prompt_parts[0], dict) and 'role' in prompt_parts[0]:
            return prompt_parts

        content: List[Dict[str, Any]] = []
        texts: List[str] = []

        for part in prompt_parts:
            try:
                if isinstance(part, str):
                    texts.append(part)
                elif hasattr(part, 'text'):
                    texts.append(str(getattr(part, 'text')))
                elif isinstance(part, dict):
                    # --- Handle image URLs
                    if 'image_url' in part:
                        content.append({"type": "image_url", "image_url": {"url": part['image_url']}})
                    # --- Handle base64 encoded image data
                    elif 'data' in part and ('mime_type' in part or 'mime' in part):
                        mime = part.get('mime_type') or part.get('mime')
                        if str(mime).startswith('image/'):
                            data_url = f"data:{mime};base64,{part['data']}"
                            content.append({"type": "image_url", "image_url": {"url": data_url}})
                    elif 'text' in part:
                        texts.append(str(part['text']))
            except Exception as e:
                logging.warning(f"Skipping malformed part in vision message preparation: {e}")
                continue

        # --- Prepend the aggregated text content to the list.
        if texts:
            content.insert(0, {"type": "text", "text": "\n\n".join(t for t in texts if t)})

        messages: List[Dict[str, Any]] = []
        if system_instruction:
            messages.append({"role": "system", "content": system_instruction})
        messages.append({"role": "user", "content": content or [{"type": "text", "text": ""}]})
        return messages

    async def generate(self, preferred_model: str, config: Any, prompt_parts: List[Any], **kwargs) -> Tuple[str, Optional[Dict[str, int]]]:
        """
        Generates a non-streaming response from an OpenRouter model.
        It includes optional headers for analytics and enforces model allowlists.
        """
        try:
            if preferred_model not in self.ALLOWED_MODELS:
                raise ValueError(f"Model '{preferred_model}' is not in the allowlist for this OpenRouter tier.")
            
            client = self._client()
            messages = self._prepare_openai_vision_messages(prompt_parts, getattr(config, 'system_instruction', None))
            
            # --- OpenRouter allows passing optional headers for analytics.
            extra_headers = {}
            if referer := os.getenv("OPENROUTER_HTTP_REFERER"):
                extra_headers["HTTP-Referer"] = referer
            if title := os.getenv("OPENROUTER_X_TITLE"):
                extra_headers["X-Title"] = title

            response = await client.chat.completions.create(
                model=preferred_model,
                messages=messages,
                extra_headers=extra_headers if extra_headers else None,
            )
            content = response.choices[0].message.content or ""
            usage = response.usage
            usage_dict = {
                "prompt_tokens": getattr(usage, 'prompt_tokens', 0),
                "completion_tokens": getattr(usage, 'completion_tokens', 0),
                "total_tokens": getattr(usage, 'total_tokens', 0)
            } if usage else None
            return content, usage_dict
        except Exception as e:
            logging.error(f"OpenRouter generate failed: {e}", exc_info=True)
            return f"Error: OpenRouter API call failed. Details: {e}", None

    async def generate_synthesis_stream(self, model: str, prompt: List[Any], **kwargs) -> AsyncGenerator[str, None]:
        """Generates a streaming response from an OpenRouter model."""
        try:
            if model not in self.ALLOWED_MODELS:
                raise ValueError(f"Model '{model}' is not in the allowlist for this OpenRouter tier.")
            
            client = self._client()
            cfg = kwargs.get('config')
            messages = self._prepare_openai_vision_messages(prompt, cfg.system_instruction if cfg else None)

            extra_headers = {}
            if referer := os.getenv("OPENROUTER_HTTP_REFERER"):
                extra_headers["HTTP-Referer"] = referer
            if title := os.getenv("OPENROUTER_X_TITLE"):
                extra_headers["X-Title"] = title

            stream = await client.chat.completions.create(
                model=model,
                messages=messages,
                stream=True,
                extra_headers=extra_headers if extra_headers else None,
            )
            async for chunk in stream:
                if content := chunk.choices[0].delta.content:
                    yield content
        except Exception as e:
            logging.error(f"OpenRouter stream failed: {e}", exc_info=True)
            yield f"Error: OpenRouter API stream failed. Details: {e}"

    @staticmethod
    def get_supported_models() -> List[str]:
        # --- This must be implemented by concrete subclasses.
        return []

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
        # --- This must be implemented by concrete subclasses.
        return []

    @staticmethod
    async def validate_key(api_key: str) -> bool:
        """Validates an OpenRouter key by listing models."""
        try:
            client = openai.AsyncOpenAI(api_key=api_key, base_url=_OpenRouterBaseProvider.BASE_URL)
            await client.models.list()
            return True
        except openai.AuthenticationError:
            logging.warning("OpenRouter key validation failed: Invalid key.")
            return False
        except Exception as e:
            logging.error(f"OpenRouter key validation failed with unexpected error: {e}")
            return False

class OpenRouterFreeProvider(_OpenRouterBaseProvider):
    """Concrete implementation for OpenRouter's free-tier models."""
    ALLOWED_MODELS = SUPPORTED_MODELS["openrouter_free"]

    @staticmethod
    def get_supported_models() -> List[str]:
        return SUPPORTED_MODELS["openrouter_free"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
        return SUPPORTED_MODELS["openrouter_free"]

class OpenRouterBillingProvider(_OpenRouterBaseProvider):
    """Concrete implementation for OpenRouter's paid-tier models."""
    ALLOWED_MODELS = SUPPORTED_MODELS["openrouter_billing"]

    @staticmethod
    def get_supported_models() -> List[str]:
        return SUPPORTED_MODELS["openrouter_billing"]

    @staticmethod
    async def list_models(api_key: str) -> List[str]:
        return SUPPORTED_MODELS["openrouter_billing"]

# -----------------------------------------------------------------------------
# --- Factory and Listing Functions
# -----------------------------------------------------------------------------

PROVIDER_MAP: Dict[str, type[LLMProvider]] = {
    "openai": OpenAIProvider,
    "anthropic": AnthropicProvider,
    "grok": GrokProvider,
    "gemini": ExternalGeminiProvider,
    "ego": EgoGeminiProvider,
    "openrouter_free": OpenRouterFreeProvider,
    "openrouter_billing": OpenRouterBillingProvider,
}

def get_llm_provider(provider_name: str, api_key: Optional[str] = None) -> LLMProvider:
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

def get_all_supported_models() -> Dict[str, List[str]]:
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
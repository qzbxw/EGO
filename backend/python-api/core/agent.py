# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import json
import logging
import os
from collections.abc import AsyncGenerator
from dataclasses import dataclass
from typing import Any, ClassVar

from pydantic import BaseModel, Field

# --- Google GenAI specific imports with exception handling
# The google-generativeai library is used for interacting with the Gemini models.
# We import specific error classes to handle API-related issues gracefully.
try:
    from google import genai
    from google.api_core import exceptions as google_exceptions
    from google.genai import errors as genai_errors
except ImportError:
    # This provides a fallback for environments where the library might not be installed,
    # preventing an immediate crash on import.
    logging.error(
        "Google GenAI library not found. Please install it using 'pip install google-genai'"
    )
    genai = None  # type: ignore
    google_exceptions = None  # type: ignore
    genai_errors = None  # type: ignore

# -----------------------------------------------------------------------------
# --- Local Module Imports
# -----------------------------------------------------------------------------
# These imports bring in project-specific components like prompts and tool definitions.
from .llm_backend import LLMProvider
from .prompts import (
    FINAL_SYNTHESIS_PROMPT_EN_AGENT,
    FINAL_SYNTHESIS_PROMPT_EN_DEEPER,
    FINAL_SYNTHESIS_PROMPT_EN_DEFAULT,
    FINAL_SYNTHESIS_PROMPT_EN_RESEARCH,
    SEQUENTIAL_THINKING_PROMPT_EN_AGENT,
    SEQUENTIAL_THINKING_PROMPT_EN_DEEPER,
    SEQUENTIAL_THINKING_PROMPT_EN_DEFAULT,
    SEQUENTIAL_THINKING_PROMPT_EN_RESEARCH,
)
from .tools import Tool

# -----------------------------------------------------------------------------
# --- Data Structures & Models
# -----------------------------------------------------------------------------


@dataclass
class ModeConfig:
    """
    A lightweight data structure to hold the configuration for an agent's mode.

    Using a dataclass is efficient here as it's an internal container for
    related data, and we don't need the overhead of Pydantic's validation.

    Attributes:
        model_name (str): The identifier for the language model to be used.
        thinking_prompt (str): The system prompt for the reasoning/thought generation phase.
        synthesis_prompt (str): The system prompt for the final response generation phase.
    """

    model_name: str
    thinking_prompt: str
    synthesis_prompt: str


class ToolCall(BaseModel):
    """
    Represents a single tool call requested by the LLM.

    This Pydantic model is used to validate the structure of the JSON output
    from the LLM when it decides to use a tool.
    """

    tool_name: str = Field(
        description="Name of the tool to be called, e.g., 'EgoSearch' or 'EgoWiki'"
    )
    tool_query: str = Field(
        description="The specific query, question, article title or code for the tool"
    )


class Thought(BaseModel):
    """
    Represents the full reasoning process of the agent in a single step.

    This model defines the structured JSON format that the LLM must follow during
    its thinking process. It provides a consistent schema for parsing the agent's
    internal state at each step.
    """

    thoughts: str = Field(
        description="A detailed thought process, including reasoning, potential code, or problem-solving steps. Include self-critique and certainty inline within the thoughts when relevant."
    )
    tool_reasoning: str = Field(
        description="If tools are needed, explain why, what kind of tool, and the specific task it should perform. If no tool is needed, this field must be an empty string."
    )
    tool_calls: list[ToolCall] = Field(
        description="A list of tool calls to be executed. If no tools are needed, this must be an empty list."
    )
    thoughts_header: str = Field(
        description="A short, general title for the thought, using a verb. For example: 'Searching for information...', 'Analyzing the request...'"
    )
    next_thought_needed: bool = Field(
        description="Set to True if another iteration of thinking is required to solve the problem, False if you have enough information to synthesize a final answer."
    )
    confidence_score: float = Field(
        default=0.5,
        description="A 0.0 to 1.0 score indicating your confidence in the current solution state. Stop only when this is high.",
    )
    self_critique: str = Field(
        default="",
        description="A specific critique of the previous action's result. Did it work? What is missing? Use this to drive the next step.",
    )
    plan_status: str = Field(
        default="in_progress",
        description="The current high-level status of the task: 'planning', 'in_progress', 'verifying', 'completed', 'failed'.",
    )


# -----------------------------------------------------------------------------
# --- Main Agent Class
# -----------------------------------------------------------------------------


class EGO:
    """
    The main class for the EGO agent.

    This agent orchestrates a multi-step reasoning process to answer complex queries.
    It generates "thoughts," uses external tools when necessary, and finally
    synthesizes a comprehensive response based on its findings.
    """

    # --- Class Attributes ---
    # Model mappings are loaded from environment variables for flexibility.
    # This allows changing the underlying models without modifying the code.
    MODEL_MAPPING: ClassVar[dict[str, str]] = {
        "default": os.getenv("GEMINI_DEFAULT_MODEL", "gemini-3-flash-preview"),
        "deeper": os.getenv("GEMINI_DEEPER_MODEL", "gemini-3-flash-preview"),
        "research": os.getenv("GEMINI_RESEARCH_MODEL", "gemini-3-flash-preview"),
        "agent": os.getenv("GEMINI_AGENT_MODEL", "gemini-3-flash-preview"),
    }
    # Defines the maximum number of retries for LLM provider calls.
    MAX_ATTEMPTS: ClassVar[int] = int(os.getenv("MAX_ATTEMPTS", 3))

    def __init__(self, backend: LLMProvider, tools: list[Tool]):
        """
        Initializes the EGO agent instance.

        Args:
            backend (LLMProvider): An instance of a language model provider that
                will be used to generate thoughts and the final response.
            tools (List[Tool]): A list of available tools that the agent can
                decide to use during its reasoning process.
        """
        self.backend = backend
        # --- Convert tool list to a dictionary for efficient O(1) name-based lookups.
        self.tools: dict[str, Tool] = {tool.name: tool for tool in tools}

        # --- A dictionary mapping modes to their specific "thinking" system prompts.
        self.THINKING_PROMPTS = {
            "default": SEQUENTIAL_THINKING_PROMPT_EN_DEFAULT,
            "deeper": SEQUENTIAL_THINKING_PROMPT_EN_DEEPER,
            "research": SEQUENTIAL_THINKING_PROMPT_EN_RESEARCH,
            "agent": SEQUENTIAL_THINKING_PROMPT_EN_AGENT,
        }

        # --- A dictionary mapping modes to their specific final "synthesis" system prompts.
        self.SYNTHESIS_PROMPTS = {
            "default": FINAL_SYNTHESIS_PROMPT_EN_DEFAULT,
            "deeper": FINAL_SYNTHESIS_PROMPT_EN_DEEPER,
            "research": FINAL_SYNTHESIS_PROMPT_EN_RESEARCH,
            "agent": FINAL_SYNTHESIS_PROMPT_EN_AGENT,
        }

    def _get_config_for_mode(self, mode: str) -> ModeConfig:
        """
        Retrieves a full configuration object for a given operational mode.

        This internal helper method selects the correct model name and prompts
        based on the provided mode. It safely falls back to the 'default'
        configuration if a specific setting for the mode is not found.

        Args:
            mode (str): The identifier for the desired mode (e.g., 'default', 'research').

        Returns:
            ModeConfig: A data object containing the complete configuration for the requested mode.
        """
        preferred_model = self.MODEL_MAPPING.get(mode, self.MODEL_MAPPING["default"])
        thinking_prompt = self.THINKING_PROMPTS.get(mode, self.THINKING_PROMPTS["default"])
        synthesis_prompt = self.SYNTHESIS_PROMPTS.get(mode, self.SYNTHESIS_PROMPTS["default"])

        return ModeConfig(
            model_name=preferred_model,
            thinking_prompt=thinking_prompt,
            synthesis_prompt=synthesis_prompt,
        )

    def _wrap_block(self, label: str, content: str) -> str:
        """
        Wraps string content in a standardized, machine-readable block.

        This formatting helps the LLM distinguish between different parts of the
        context (e.g., chat history vs. tool output). The function is idempotent,
        meaning it won't add nested wrappers if the content is already wrapped.

        Args:
            label (str): A descriptive, uppercase label (e.g., "CHAT HISTORY").
            content (str): The text content to be wrapped.

        Returns:
            str: The formatted content block, or an empty string if the input was empty.
                Example:
                "[BEGIN CHAT HISTORY MARKDOWN]
                User: Hello
                Agent: Hi
                [END CHAT HISTORY]"
        """
        if not content:
            return ""
        # --- Check for exact wrapper pattern to avoid re-wrapping properly formatted blocks.
        begin_marker = f"[BEGIN {label} MARKDOWN]"
        end_marker = f"[END {label}]"
        if content.startswith(begin_marker) and content.endswith(end_marker):
            return content
        return f"[BEGIN {label} MARKDOWN]\n{content}\n[END {label}]"

    async def _process_file_with_llm(self, file_content: str, file_name: str, query: str) -> str:
        """
        Uses the LLM to analyze the content of a single file in relation to a query.

        This method formulates a prompt that includes the file's content and the user's
        query, then asks the LLM to provide an answer based on the file. It includes
        error handling for the API call.

        Args:
            file_content (str): The text content of the file.
            file_name (str): The name of the file, used for context.
            query (str): The user's query to be answered based on the file.

        Returns:
            str: The LLM's response, or an error message if processing failed.
        """
        try:
            # --- Construct a clear prompt for the LLM.
            prompt = f"""
            Here is the content of the file named '{file_name}':
            --- FILE CONTENT BEGIN ---
            {file_content}
            --- FILE CONTENT END ---

            Based on the content of this file, please answer the following question: {query}
            """
            # --- Use a fast model for this focused task.
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash", config={}, prompt_parts=[prompt]
            )
            return response_text
        except Exception as e:
            # --- Log the error with traceback for debugging.
            logging.error(f"Error processing file '{file_name}' with LLM: {e}", exc_info=True)
            return f"Error: Could not process the file '{file_name}'. Details: {e}"

    async def _summarize_block(
        self, model: str, label: str, content: str, target_chars: int = 2000
    ) -> str:
        """
        Asynchronously summarizes a text block to reduce its length.

        This is a crucial utility for managing context window size. It asks the LLM
        to create a concise summary. If the summarization call fails, it falls back
        to a simple truncation method to ensure the process doesn't halt.

        Args:
            model (str): The model to use for summarization.
            label (str): A label for the content block (e.g., "CHAT HISTORY").
            content (str): The text content to be summarized.
            target_chars (int, optional): The target character length for the summary.

        Returns:
            str: A wrapped block containing the summary or truncated text.

        Raises:
            Exception: Re-raises any unexpected exceptions after logging them.
        """
        if not content:
            return ""

        # --- Optimization: If content is already small enough (within 120% of target),
        # --- just use it as is to save an LLM call and latency.
        if len(content) <= target_chars * 1.2:
            return self._wrap_block(label, content)

        try:
            # --- System instruction for the summarization task.
            sys_inst = (
                "You are a text compressor. Summarize the given block, preserving key facts, "
                "names, numbers, tool results, decisions, and action items."
                f" The output must be less than or equal to {target_chars} characters."
                " Use concise Markdown."
            )
            prompt_parts = [f"[BLOCK TO COMPRESS - {label}]\n{content}\n[END BLOCK]"]

            # --- Configure the generation for a factual, low-temperature summary.
            config = genai.types.GenerateContentConfig(temperature=0.2)
            config.system_instruction = sys_inst

            response_text, _ = await self.backend.generate(
                preferred_model=model, config=config, prompt_parts=prompt_parts
            )
            compressed = response_text.strip()

            # --- Enforce a hard cap if the model exceeded the target length.
            if len(compressed) > target_chars:
                compressed = compressed[:target_chars]
            return self._wrap_block(label, compressed)

        except (
            genai_errors.ClientError,
            genai_errors.ServerError,
            google_exceptions.GoogleAPICallError,
        ) as e:
            # --- Handle known API errors gracefully with a fallback.
            logging.warning(
                f"API call for summarization failed for block '{label}'. Error: {e}. "
                "Falling back to content truncation."
            )
            # --- Fallback: combine the beginning and end of the content.
            head = content[: target_chars // 2]
            tail = content[-target_chars // 2 :]
            fallback = head + "\n...\n[Content Truncated]\n...\n" + tail
            return self._wrap_block(label, fallback)

        except Exception as e:
            # --- Log unexpected errors and re-raise to avoid silent failures.
            logging.error(f"An unexpected error occurred during summarization: {e}", exc_info=True)
            raise

    async def generate_thought(
        self,
        query: str,
        mode: str,
        chat_history: str,
        thoughts_history: str,
        custom_instructions: str | None,
        prompt_parts_from_files: list[Any],
        model: str | None = None,
        vector_memory=None,
        user_id: str | None = None,
        session_uuid: str | None = None,
        current_log_id: int | None = None,
        memory_enabled: bool = True,
        current_plan: Any | None = None,
        current_date: str | None = None,
        user_profile: str | None = None,
    ) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
        """
        Generates a single structured thought for the agent's reasoning process.

        This core function constructs a detailed prompt, calls the LLM to generate
        a `Thought` object in JSON format, and handles failures with a retry-and-shrink
        strategy. If parsing the LLM's JSON response fails, it returns a safe,
        structured error message.

        Args:
            query (str): The user's latest query.
            mode (str): The operational mode for the agent (e.g., 'default', 'research').
            chat_history (str): The history of the conversation.
            thoughts_history (str): The history of the agent's previous thoughts.
            custom_instructions (Optional[str]): User-provided instructions.
            prompt_parts_from_files (List[Any]): Content parts (e.g., images, text from files).
            model (Optional[str]): An override for the mode's default model.

        Returns:
            A tuple containing:
            - The generated thought as a dictionary, or a fallback object on failure.
            - The token usage metadata from the API call, or None on failure.
        """
        mode_config = self._get_config_for_mode(mode)
        if model:
            mode_config.model_name = model

        # --- Inject Current Date ---
        final_custom_instructions = custom_instructions or "None."
        if current_date:
            final_custom_instructions = (
                f"Current Date: {current_date}.\n{final_custom_instructions}"
            )

        # --- Inject User Profile ---
        if user_profile:
            final_custom_instructions = (
                f"{final_custom_instructions}\n\n[USER PROFILE CONTEXT]\n{user_profile}"
            )

        # --- Isolate and process file parts before constructing the main prompt.
        image_parts = []
        file_processing_results = []
        if prompt_parts_from_files:
            for part in prompt_parts_from_files:
                # --- This check assumes a custom structure for file parts.
                if isinstance(part, dict) and part.get("type") == "file":
                    result = await self._process_file_with_llm(part["content"], part["name"], query)
                    file_processing_results.append(result)
                else:
                    image_parts.append(part)

        # --- Append file processing results to the thoughts history.
        if file_processing_results:
            file_results_block = self._wrap_block(
                "FILE ANALYSIS RESULTS", "\n".join(file_processing_results)
            )
            thoughts_history = (
                f"{thoughts_history}\n{file_results_block}"
                if thoughts_history
                else file_results_block
            )

        # --- Inject relevant memory context if available
        retrieved_snippets_text = ""
        if vector_memory and user_id and memory_enabled:
            try:
                memory_texts = await vector_memory.search_for_injection(
                    user_id=user_id,
                    query=query,
                    top_k=5,  # Increased for better context
                    session_id=session_uuid,
                    current_log_id=current_log_id,
                )
                if memory_texts:
                    retrieved_snippets_text = "\n".join(memory_texts)
                    logging.info(
                        f"Injected {len(memory_texts)} memory contexts for user '{user_id}'"
                    )
            except Exception as e:
                logging.warning(f"Failed to inject memory context: {e}")

        # --- Format Current Plan if available
        plan_text = ""
        if current_plan:
            try:
                steps_str = []
                # Handle both object and dict (Pydantic model vs dict)
                steps = (
                    current_plan.steps
                    if hasattr(current_plan, "steps")
                    else current_plan.get("steps", [])
                )
                title = (
                    current_plan.title
                    if hasattr(current_plan, "title")
                    else current_plan.get("title", "Unknown Plan")
                )

                for step in steps:
                    s_desc = (
                        step.description
                        if hasattr(step, "description")
                        else step.get("description", "")
                    )
                    s_status = (
                        step.status if hasattr(step, "status") else step.get("status", "pending")
                    )
                    s_order = (
                        step.step_order
                        if hasattr(step, "step_order")
                        else step.get("step_order", 0)
                    )

                    status_mark = "[ ]"
                    if s_status == "completed":
                        status_mark = "[X]"
                    elif s_status == "in_progress":
                        status_mark = "[>]"
                    elif s_status == "failed":
                        status_mark = "[!]"
                    elif s_status == "skipped":
                        status_mark = "[-]"

                    steps_str.append(f"{status_mark} {s_order}. {s_desc} ({s_status})")

                plan_text = f"[ACTIVE MISSION PLAN: {title}]\n" + "\n".join(steps_str)
            except Exception as e:
                logging.warning(f"Failed to format current plan: {e}")

        if chat_history and not chat_history.strip().startswith("[BEGIN"):
            chat_history = chat_history.strip()

        # --- Process thoughts_history: convert to simple text format
        processed_thoughts = ""
        if thoughts_history and thoughts_history.strip() and thoughts_history.strip() != "null":
            if thoughts_history.strip().startswith("["):
                try:
                    thoughts_json = json.loads(thoughts_history)
                    logging.info(
                        f"[THOUGHTS PARSING] Parsed {len(thoughts_json)} items from thoughts_history"
                    )
                    if isinstance(thoughts_json, list):
                        markdown_thoughts = []
                        for i, item in enumerate(thoughts_json, 1):
                            if isinstance(item, dict):
                                item_type = item.get("type")
                                # Tool results might have type 'tool_output' or 'tool_error' or just be the result
                                if item_type in ["tool_output", "tool_result"]:
                                    tool_name = item.get("tool_name", "Unknown Tool")
                                    output = item.get("output", "") or item.get("result", "")
                                    markdown_thoughts.append(f"### {tool_name} Result\n{output}")
                                elif item_type == "tool_error":
                                    tool_name = item.get("tool_name", "Unknown Tool")
                                    error = item.get("error", "Unknown error")
                                    markdown_thoughts.append(f"### {tool_name} Error\n{error}")
                                else:
                                    # Regular thought or unknown type
                                    content = (
                                        item.get("content")
                                        or item.get("thoughts")
                                        or item.get("text", "")
                                    )
                                    header = item.get("thoughts_header") or item.get(
                                        "header", f"Thought {i}"
                                    )
                                    reasoning = item.get("tool_reasoning", "")
                                    confidence = item.get("confidence_score")
                                    critique = item.get("self_critique")
                                    status = item.get("plan_status")

                                    if content:
                                        thought_text = f"### {header}\n{content}"
                                        if reasoning:
                                            thought_text += f"\n\n**Tool Reasoning:** {reasoning}"
                                        if critique:
                                            thought_text += f"\n\n**Self Critique:** {critique}"

                                        meta_parts = []
                                        if confidence is not None:
                                            meta_parts.append(f"Confidence: {confidence}")
                                        if status:
                                            meta_parts.append(f"Status: {status}")

                                        if meta_parts:
                                            thought_text += f"\n\n_({' | '.join(meta_parts)})_"

                                        markdown_thoughts.append(thought_text)
                        processed_thoughts = "\n\n".join(markdown_thoughts)
                        logging.info(
                            f"[THOUGHTS PARSING] Final processed_thoughts length: {len(processed_thoughts)}"
                        )
                except (json.JSONDecodeError, KeyError) as e:
                    logging.warning(f"Failed to parse thoughts_history as JSON: {e}")
                    processed_thoughts = thoughts_history
            else:
                processed_thoughts = thoughts_history

        logging.debug(
            f"chat_history length: {len(chat_history)}, processed_thoughts length: {len(processed_thoughts)}"
        )

        full_prompt = mode_config.thinking_prompt.format(
            custom_instructions=final_custom_instructions,
            chat_history=chat_history,
            thoughts_history=processed_thoughts,
            user_query=query,
            retrieved_snippets=retrieved_snippets_text,
            user_profile=user_profile or "Not available yet.",
        )

        # Inject plan if exists
        if plan_text:
            full_prompt = f"{plan_text}\n\n{full_prompt}"

        # Put everything in prompt_parts for better model understanding
        prompt_parts = [*list(image_parts or []), full_prompt]
        # --- Configure the generation to expect a JSON object matching the Thought schema.
        generation_config = genai.types.GenerateContentConfig(
            temperature=0.7, response_mime_type="application/json", response_schema=Thought
        )

        # --- Main loop for API calls with retry-and-shrink logic.
        for attempt in range(self.MAX_ATTEMPTS):
            try:
                response_text, usage_metadata = await self.backend.generate(
                    preferred_model=mode_config.model_name,
                    config=generation_config,
                    prompt_parts=prompt_parts,
                )

                # --- Safely parse the JSON response.
                try:
                    parsed_json = json.loads(response_text)
                    print(f"[DEBUG] Parsed JSON response: {parsed_json}")
                    print(
                        f"[DEBUG] thoughts_header in response: {parsed_json.get('thoughts_header', 'NOT FOUND')}"
                    )
                    return parsed_json, usage_metadata
                except json.JSONDecodeError as je:
                    logging.warning(
                        f"Non-JSON response for generate_thought. Error: {je}. Text: {response_text[:200]}"
                    )
                    fallback_thought = Thought(
                        thoughts="The model's response was not valid JSON. A safe fallback was generated. Note: provider returned unparsable text.",
                        tool_reasoning="",
                        tool_calls=[],
                        thoughts_header="JSON Parsing Error",
                        next_thought_needed=False,
                        confidence_score=0.0,
                        self_critique="Failed to parse output.",
                        plan_status="failed",
                    )
                    return json.loads(fallback_thought.model_dump_json()), usage_metadata

            except (
                genai_errors.ClientError,
                genai_errors.ServerError,
                google_exceptions.GoogleAPICallError,
            ) as e:
                logging.warning(
                    f"generate_thought (Attempt {attempt + 1}/{self.MAX_ATTEMPTS}) failed: {e}. Retrying with smaller context..."
                )

                if attempt >= self.MAX_ATTEMPTS - 1:
                    break  # Last attempt failed, exit loop.

                # --- Context Reduction Logic: Summarize histories to save tokens.
                chat_summary = await self._summarize_block(
                    mode_config.model_name, "CHAT HISTORY", chat_history, target_chars=1600
                )
                thoughts_summary = await self._summarize_block(
                    mode_config.model_name, "THOUGHTS HISTORY", thoughts_history, target_chars=1600
                )

                # --- Rebuild the system instruction with the summarized histories.
                system_instruction = mode_config.thinking_prompt.format(
                    custom_instructions=final_custom_instructions,
                    chat_history=chat_summary,
                    thoughts_history=thoughts_summary,
                    user_query=query,
                    retrieved_snippets=retrieved_snippets_text,
                )

                if plan_text:
                    system_instruction = f"{plan_text}\n\n{system_instruction}"

                generation_config.system_instruction = system_instruction
                continue

        # --- This block is reached only after all retries have failed.
        logging.error(
            f"All {self.MAX_ATTEMPTS} attempts to generate a thought failed for query: '{query[:100]}...'"
        )
        fallback_thought = Thought(
            thoughts="Failed to generate a thought after multiple context reduction attempts. Likely due to excessive context size or a persistent API error.",
            tool_reasoning="",
            tool_calls=[],
            thoughts_header="Processing Error",
            nextThoughtNeeded=False,
            confidence_score=0.0,
            self_critique="Persistent API failure.",
            plan_status="failed",
        )
        return json.loads(fallback_thought.model_dump_json()), None

    async def synthesize_stream(
        self,
        query: str,
        mode: str,
        chat_history: str,
        thoughts_history: str,
        custom_instructions: str | None,
        prompt_parts_from_files: list[Any],
        model: str | None = None,
        vector_memory=None,
        user_id: str | None = None,
        session_uuid: str | None = None,
        current_log_id: int | None = None,
        memory_enabled: bool = True,
        current_plan: Any | None = None,
        current_date: str | None = None,
        user_profile: str | None = None,
    ) -> AsyncGenerator[str | dict[str, Any], None]:
        """
        Generates the final user-facing response as an asynchronous stream.

        This function takes the complete context (histories, thoughts, files) and
        streams a polished answer from the LLM. It includes the same retry-and-shrink
        logic as `generate_thought` to handle context size errors gracefully.

        Args:
            query (str): The user's latest query.
            mode (str): The operational mode for the agent.
            chat_history (str): The history of the conversation.
            thoughts_history (str): The history of the agent's thoughts.
            custom_instructions (Optional[str]): User-provided instructions.
            prompt_parts_from_files (List[Any]): Content parts from files.
            model (Optional[str]): An override for the mode's default model.

        Yields:
            An asynchronous generator that yields response chunks (tokens) as strings
            or control dictionaries.
        """
        mode_config = self._get_config_for_mode(mode)
        if model:
            mode_config.model_name = model

        # --- Inject Current Date ---
        final_custom_instructions = custom_instructions or "None."
        if current_date:
            final_custom_instructions = (
                f"Current Date: {current_date}.\n{final_custom_instructions}"
            )

        # --- Inject User Profile ---
        if user_profile:
            final_custom_instructions = (
                f"{final_custom_instructions}\n\n[USER PROFILE CONTEXT]\n{user_profile}"
            )

        # --- Inject relevant memory context if available
        retrieved_snippets_text = ""
        if vector_memory and user_id and memory_enabled:
            try:
                memory_texts = await vector_memory.search_for_injection(
                    user_id=user_id,
                    query=query,
                    top_k=5,
                    session_id=session_uuid,
                    current_log_id=current_log_id,
                )
                if memory_texts:
                    retrieved_snippets_text = "\n".join(memory_texts)
                    logging.info(
                        f"Injected {len(memory_texts)} memory contexts for synthesis for user '{user_id}'"
                    )
            except Exception as e:
                logging.warning(f"Failed to inject memory context for synthesis: {e}")

        # --- Format Current Plan if available
        plan_text = ""
        if current_plan:
            try:
                steps_str = []
                # Handle both object and dict
                steps = (
                    current_plan.steps
                    if hasattr(current_plan, "steps")
                    else current_plan.get("steps", [])
                )
                title = (
                    current_plan.title
                    if hasattr(current_plan, "title")
                    else current_plan.get("title", "Unknown Plan")
                )

                for step in steps:
                    s_desc = (
                        step.description
                        if hasattr(step, "description")
                        else step.get("description", "")
                    )
                    s_status = (
                        step.status if hasattr(step, "status") else step.get("status", "pending")
                    )
                    s_order = (
                        step.step_order
                        if hasattr(step, "step_order")
                        else step.get("step_order", 0)
                    )

                    status_mark = "[ ]"
                    if s_status == "completed":
                        status_mark = "[X]"
                    elif s_status == "in_progress":
                        status_mark = "[>]"
                    elif s_status == "failed":
                        status_mark = "[!]"
                    elif s_status == "skipped":
                        status_mark = "[-]"

                    steps_str.append(f"{status_mark} {s_order}. {s_desc} ({s_status})")

                plan_text = f"[ACTIVE MISSION PLAN: {title}]\n" + "\n".join(steps_str)
            except Exception as e:
                logging.warning(f"Failed to format current plan for synthesis: {e}")

        if chat_history and not chat_history.strip().startswith("[BEGIN"):
            chat_history = chat_history.strip()

        # --- Process thoughts_history: convert to simple text format
        processed_thoughts = ""
        if thoughts_history and thoughts_history.strip() and thoughts_history.strip() != "null":
            if thoughts_history.strip().startswith("["):
                try:
                    thoughts_json = json.loads(thoughts_history)
                    logging.info(
                        f"[SYNTHESIS THOUGHTS PARSING] Parsed {len(thoughts_json)} items from thoughts_history"
                    )
                    if isinstance(thoughts_json, list):
                        markdown_thoughts = []
                        for i, thought in enumerate(thoughts_json, 1):
                            if isinstance(thought, dict):
                                thought_type = thought.get("type", "unknown")
                                logging.debug(
                                    f"[SYNTHESIS THOUGHTS PARSING] Item {i}: type={thought_type}, keys={list(thought.keys())}"
                                )

                                # Handle tool outputs specifically
                                if thought_type == "tool_output":
                                    tool_name = thought.get("tool_name", "Unknown Tool")
                                    output = thought.get("output", "")
                                    logging.info(
                                        f"[SYNTHESIS THOUGHTS PARSING] Processing tool_output: {tool_name}, output length: {len(str(output))}"
                                    )
                                    if output:
                                        markdown_thoughts.append(f"## {tool_name} Result\n{output}")
                                    else:
                                        markdown_thoughts.append(
                                            f"## {tool_name} Result\n[Tool returned empty response]"
                                        )

                                # Handle tool errors specifically
                                elif thought_type == "tool_error":
                                    tool_name = thought.get("tool_name", "Unknown Tool")
                                    error_msg = thought.get("error", "Unknown error")
                                    logging.warning(
                                        f"[SYNTHESIS THOUGHTS PARSING] Processing tool_error: {tool_name}, error: {error_msg}"
                                    )
                                    markdown_thoughts.append(f"## {tool_name} Error\n{error_msg}")

                                # Handle regular thoughts
                                else:
                                    content = (
                                        thought.get("content")
                                        or thought.get("thoughts")
                                        or thought.get("text", "")
                                    )
                                    header = thought.get("thoughts_header", f"Thought {i}")
                                    reasoning = thought.get("tool_reasoning", "")
                                    confidence = thought.get("confidence_score")
                                    critique = thought.get("self_critique")
                                    status = thought.get("plan_status")

                                    if content:
                                        logging.debug(
                                            f"[SYNTHESIS THOUGHTS PARSING] Processing thought: header={header}, content length: {len(str(content))}"
                                        )
                                        thought_text = f"## {header}\n{content}"
                                        if reasoning:
                                            thought_text += f"\n\n**Tool Reasoning:** {reasoning}"
                                        if critique:
                                            thought_text += f"\n\n**Self Critique:** {critique}"

                                        meta_parts = []
                                        if confidence is not None:
                                            meta_parts.append(f"Confidence: {confidence}")
                                        if status:
                                            meta_parts.append(f"Status: {status}")

                                        if meta_parts:
                                            thought_text += f"\n\n_({' | '.join(meta_parts)})_"

                                        markdown_thoughts.append(thought_text)
                        processed_thoughts = "\n\n".join(markdown_thoughts)
                        logging.info(
                            f"[SYNTHESIS THOUGHTS PARSING] Final processed_thoughts length: {len(processed_thoughts)}"
                        )
                except (json.JSONDecodeError, KeyError) as e:
                    logging.warning(f"[SYNTHESIS] Failed to parse thoughts_history as JSON: {e}")
                    processed_thoughts = thoughts_history
            else:
                processed_thoughts = thoughts_history

        full_prompt = mode_config.synthesis_prompt.format(
            custom_instructions=final_custom_instructions,
            chat_history=chat_history,
            thoughts_history=processed_thoughts,
            user_query=query,
            retrieved_snippets=retrieved_snippets_text,
        )

        if retrieved_snippets_text:
            full_prompt = f"[RELEVANT PAST CONTEXT]\n{retrieved_snippets_text}\n[END CONTEXT]\n\n{full_prompt}"

        # Inject plan
        if plan_text:
            full_prompt = f"{plan_text}\n\n{full_prompt}"

        # Put everything in prompt_parts for better model understanding
        prompt_parts = [*list(prompt_parts_from_files or []), full_prompt]

        generation_config = genai.types.GenerateContentConfig(temperature=0.8)

        # --- Main loop for API calls with retry-and-shrink logic.
        for attempt in range(self.MAX_ATTEMPTS):
            has_yielded = False
            try:
                # --- Use 'async for' to iterate over the streamed response chunks.
                async for chunk in self.backend.generate_synthesis_stream(
                    model=mode_config.model_name, config=generation_config, prompt=prompt_parts
                ):
                    yield chunk
                    has_yielded = True
                return  # --- Gracefully exit after successful stream completion.

            except (
                genai_errors.ClientError,
                genai_errors.ServerError,
                google_exceptions.GoogleAPICallError,
            ) as e:
                logging.warning(
                    f"synthesize_stream (Attempt {attempt + 1}/{self.MAX_ATTEMPTS}) failed: {e}. Retrying with smaller context..."
                )

                if attempt >= self.MAX_ATTEMPTS - 1:
                    break

                if has_yielded:
                    # We partially streamed a response, but then failed.
                    # We are about to retry, which means a new stream will start.
                    # We must inform the consumer to clear the partial output.
                    yield {"type": "reset"}

                # --- Context Reduction Logic.
                chat_summary = await self._summarize_block(
                    mode_config.model_name, "CHAT HISTORY", chat_history, target_chars=1600
                )
                thoughts_summary = await self._summarize_block(
                    mode_config.model_name, "THOUGHTS HISTORY", thoughts_history, target_chars=1600
                )

                system_instruction = mode_config.synthesis_prompt.format(
                    custom_instructions=final_custom_instructions,
                    chat_history=chat_summary,
                    thoughts_history=thoughts_summary,
                    user_query=query,
                    retrieved_snippets=retrieved_snippets_text,
                )

                if plan_text:
                    system_instruction = f"{plan_text}\n\n{system_instruction}"

                generation_config.system_instruction = system_instruction
                continue

        # --- This block is reached only after all retries have failed.
        logging.error(
            f"All {self.MAX_ATTEMPTS} attempts to synthesize a response failed for query: '{query[:100]}...'"
        )
        yield "\n\n[I apologize, but I encountered a persistent error while trying to generate a response. Please try again later.]"
        return

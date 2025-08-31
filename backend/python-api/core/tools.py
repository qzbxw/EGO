# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import asyncio
import logging
import re
from datetime import datetime
from typing import Optional

# --- Third-party libraries for specific tools
import sympy
import wikipediaapi

# --- Google GenAI specific imports for tool functionality and error handling
try:
    from google import genai
    from google.genai import errors as genai_errors
except ImportError:
    logging.critical("Google GenAI library not found. Please install 'google-genai'.")
    genai = None
    genai_errors = None

# -----------------------------------------------------------------------------
# --- Local Module Imports
# -----------------------------------------------------------------------------
from .llm_backend import LLMProvider
from .memory_db import VectorMemory
from .prompts import (
    ALTER_EGO_PROMPT_EN,
    CODE_EXEC_PROMPT_EN,
    EGO_SEARCH_PROMPT_EN,
    EGO_TUBE_PROMPT_EN,
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

    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Executes the tool's primary function with a given query.

        This method must be implemented by all subclasses.

        Args:
            query (str): The input query, command, or data for the tool.
            user_id (Optional[str]): The user's unique identifier, which is
                required for tools that operate on user-specific data, such
                as memory. Defaults to None.

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
            name="EgoSearch",
            desc="Performs a Google Search to find real-time information, news, and facts on the web."
        )
        self.backend = backend

    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Executes a Google Search using the Gemini model's built-in tool.

        Args:
            query: The search term or question.
            user_id: Not used by this tool.

        Returns:
            A string containing the search results, or an error message.
        """
        logging.info(f"--- EgoSearch: Executing with query: '{query}' ---")
        
        # --- Configure the model to use its native Google Search capability.
        search_tool = genai.types.Tool(google_search=genai.types.GoogleSearch())
        config = genai.types.GenerateContentConfig(
            temperature=0.1,
            tools=[search_tool],
            system_instruction=EGO_SEARCH_PROMPT_EN
        )
        
        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash-lite", 
                config=config,
                prompt_parts=[query]
            )
            return response_text
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"EgoSearch failed for query '{query}'. API Error: {e}")
            return "Search is temporarily unavailable due to a technical issue. I will proceed without it or you can ask me to try again later."

class EgoTube(Tool):
    """A tool that uses Gemini to analyze and get information from a YouTube video URL."""
    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="EgoTube",
            desc="Analyzes a YouTube video from its URL to answer questions or provide summaries. Example: 'Summarize this video https://www.youtube.com/watch?v=some_id'"
        )
        self.backend = backend

    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Extracts a YouTube URL from the query and uses Gemini to analyze the video.

        Args:
            query: A string containing a YouTube URL and an optional question/command.
            user_id: Not used by this tool.

        Returns:
            The model's analysis of the video, or an error message if no URL is found.
        """
        logging.info(f"--- EgoTube: Received query: '{query}' ---")

        # --- Use regex to find YouTube URLs (both standard and short formats).
        url_match = re.search(r'https?://(?:www\.youtube\.com/watch\?v=|youtu\.be/)[\w-]+', query)

        if not url_match:
            logging.warning(f"EgoTube: No YouTube URL found in query: '{query}'")
            return "Error: No YouTube URL was found in your request. Please provide a valid link to a YouTube video."

        video_url = url_match.group(0)
        # --- The prompt is whatever is left of the query after removing the URL.
        prompt_text = query.replace(video_url, '').strip() or "Briefly summarize the content of this video."

        # --- Prepare the multi-modal prompt for the Gemini API.
        video_part = genai.types.Part(file_data=genai.types.FileData(file_uri=video_url))
        text_part = genai.types.Part(text=prompt_text)
        
        config = genai.types.GenerateContentConfig(
            temperature=0.1,
            system_instruction=EGO_TUBE_PROMPT_EN
        )

        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash-lite",
                config=config,
                prompt_parts=[video_part, text_part]
            )
            return response_text
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"EgoTube failed for URL '{video_url}'. API Error: {e}")
            return "Video analysis is currently unavailable due to a technical issue. Please try again later."

class AlterEgo(Tool):
    """A tool that uses a different persona/model to re-analyze a thought or query."""
    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="AlterEgo",
            desc="Engages a creative and unconventional persona named 'AlterEgo' to analyze a thought or query from a completely different perspective."
        )
        self.backend = backend
    
    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Invokes a different model persona to get a creative or alternative viewpoint.

        Args:
            query: The thought or question to be re-analyzed.
            user_id: Not used by this tool.

        Returns:
            A response from the AlterEgo persona.
        """
        logging.info(f"--- AlterEgo: Engaging with query: '{query}' ---")
        
        # --- High temperature for more creative and diverse responses.
        config = genai.types.GenerateContentConfig(
            temperature=0.9,
            system_instruction=ALTER_EGO_PROMPT_EN
        )
        
        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash", 
                config=config,
                prompt_parts=[query]
            )
            logging.info("--- AlterEgo: Response successfully generated. ---")
            return response_text
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"AlterEgo failed for query '{query}'. API Error: {e}")
            return "AlterEgo is temporarily unavailable. Reverting to my standard mode of thinking."

class EgoCodeExec(Tool):
    """A tool that executes Python code using the model's built-in, secure code execution environment."""
    def __init__(self, backend: LLMProvider):
        super().__init__(
            name="EgoCodeExec",
            desc="Executes Python code in a secure, sandboxed environment via the model's native capabilities. Useful for complex calculations, data manipulation, and algorithm execution."
        )
        self.backend = backend
        
    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Sends Python code to the Gemini model for secure execution.

        Args:
            query: A string containing the Python code to be executed.
            user_id: Not used by this tool.

        Returns:
            The output of the executed code (stdout) or an error message.
        """
        logging.info("--- EgoCodeExec: Executing provided code snippet. ---")
        
        # --- Configure the model to use its native Code Execution tool.
        code_execution_tool = genai.types.Tool(code_execution=genai.types.ToolCodeExecution())
        config = genai.types.GenerateContentConfig(
            temperature=0.1,
            system_instruction=CODE_EXEC_PROMPT_EN
        )
        
        try:
            response_text, _ = await self.backend.generate(
                preferred_model="gemini-2.5-flash-lite", 
                config=config,
                prompt_parts=[query],
                tools=[code_execution_tool]
            )
            return response_text
        except (genai_errors.ClientError, genai_errors.ServerError) as e:
            logging.warning(f"EgoCodeExec failed to execute. API Error: {e}")
            return "Code execution is temporarily unavailable. I can provide the algorithm or pseudocode instead."

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
            name="EgoMemory",
            desc="Searches a vector memory to find relevant information from the user's past conversations."
        )
        self.vector_memory = vector_memory

    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Performs a semantic search on the user's long-term memory.

        Args:
            query: The search query to find relevant memories.
            user_id: The unique ID of the user whose memory is being searched. This is required.

        Returns:
            A formatted string of relevant memories found, or a message indicating no results.
        """
        logging.info(f"--- EgoMemory: Searching for user '{user_id}' with query: '{query}' ---")
        
        if not user_id:
            logging.warning("EgoMemory: A search was attempted without a user_id.")
            return "Error: user_id was not provided, so I cannot access your memory."
        
        try:
            # --- Perform the search using the provided VectorMemory instance.
            hits = await self.vector_memory.search(user_id, query, top_k=5)
        except Exception as e:
            logging.error(f"EgoMemory search failed for user '{user_id}'. Error: {e}", exc_info=True)
            return "An unexpected error occurred while searching my memory banks."
            
        if not hits:
            return "No relevant information was found in my memory for your query."
        
        # --- Format the retrieved memories for the LLM.
        response_lines = ["I found the following relevant snippets in my memory:"]
        for hit in hits:
            try:
                # --- Safely parse and format the timestamp.
                dt = datetime.fromisoformat(hit.created_at.replace('Z', '+00:00'))
                timestamp = f" (from: {dt.strftime('%Y-%m-%d %H:%M')})"
            except (ValueError, TypeError, AttributeError):
                timestamp = "" # --- Ignore if timestamp is missing or malformed.
            
            response_lines.append(f"- \"{hit.text}\" (relevance: {hit.score:.2f}){timestamp}")
        
        return "\n".join(response_lines)

class EgoCalc(Tool):
    """A tool that performs mathematical calculations using the SymPy library for safety and accuracy."""
    def __init__(self):
        super().__init__(
            name="EgoCalc",
            desc="Performs precise mathematical calculations. Accepts expressions like 'sqrt(8) + 5**3'."
        )
        
    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Safely evaluates a mathematical expression using SymPy.

        Args:
            query: The mathematical expression to evaluate (e.g., "1/3 + pi").
            user_id: Not used by this tool.

        Returns:
            The result of the calculation or a detailed error message.
        """
        logging.info(f"--- EgoCalc: Evaluating expression: '{query}' ---")
        try:
            # --- Use sympy.sympify for safe evaluation, avoiding the dangers of eval().
            # 'strict=True' ensures that strings like "_i" are not interpreted as symbols.
            expr = sympy.sympify(query, strict=True)
            result = expr.evalf() # --- .evalf() evaluates the expression to a floating-point number.
            return f"Result: {result}"
        except (sympy.SympifyError, SyntaxError) as e:
            logging.warning(f"EgoCalc failed to parse query '{query}'. Error: {e}")
            return f"Error: The mathematical expression could not be parsed. Please check the syntax. Details: {e}"
        except Exception as e:
            logging.error(f"EgoCalc encountered an unexpected error for query '{query}': {e}", exc_info=True)
            return "An unexpected error occurred during the calculation."

class EgoWiki(Tool):
    """A tool that uses the Wikipedia API to look up factual information."""
    def __init__(self):
        super().__init__(
            name="EgoWiki",
            desc="Uses Wikipedia to search for precise information, definitions, and summaries of articles."
        )
        # --- Initialize the API client with a descriptive user agent.
        self.wiki_wiki = wikipediaapi.Wikipedia(
            user_agent="EGO Agent Knowledge Retrieval (ego-agent/1.0)",
            language="en"
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
                # --- Return the first section (~500 chars) for brevity in the agent's context.
                summary = wiki_page.summary
                return summary[:500] + ('...' if len(summary) > 500 else '')
            else:
                return f"The page '{query}' was not found on Wikipedia. Please try a different title."
        except Exception as e:
            logging.error(f"EgoWiki API call failed for query '{query}'. Error: {e}", exc_info=True)
            return 'An external error occurred while trying to contact the Wikipedia API.'

    async def use(self, query: str, user_id: Optional[str] = None) -> str:
        """
        Searches for a Wikipedia article in a non-blocking way.

        Args:
            query: The article title to look up.
            user_id: Not used by this tool.

        Returns:
            A summary of the Wikipedia article.
        """
        logging.info(f"--- EgoWiki: Searching for article: '{query}' ---")
        # --- Run the synchronous network call in a separate thread to avoid blocking
        # --- the main async event loop. This is crucial for async performance.
        loop = asyncio.get_running_loop()
        return await loop.run_in_executor(None, self._search_wiki_sync, query)
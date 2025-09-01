# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import json
import os
import io
from typing import List, Optional, Any, AsyncGenerator
from contextlib import asynccontextmanager
from tempfile import NamedTemporaryFile

# --- Third-party libraries for specific functionalities
import aioboto3
import fitz  # PyMuPDF, used for PDF processing
from botocore.config import Config
from fastapi import FastAPI, Form, File, UploadFile, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse, JSONResponse, PlainTextResponse
from pydantic import BaseModel, Field
from google.genai.types import Part
from PIL import Image

# -----------------------------------------------------------------------------
# --- Local Module Imports
# -----------------------------------------------------------------------------
from core.agent import EGO
from core.memory_db import VectorMemory
from core.llm_backend import get_llm_provider, EgoGeminiProvider
from core.tools import EgoSearch, EgoCalc, EgoWiki, AlterEgo, EgoCodeExec, EgoTube, EgoMemory
from utils.logger import setup_logging, get_logger

# -----------------------------------------------------------------------------
# --- Application Setup
# -----------------------------------------------------------------------------
# --- This must be called at the very beginning to configure the logging system.
setup_logging()
# --- Get a logger instance for this main module.
log = get_logger(__name__)

# -----------------------------------------------------------------------------
# --- Constants
# -----------------------------------------------------------------------------
# --- Defines the maximum total size of all files combined in a single request.
MAX_COMBINED_FILE_BYTES = 100 * 1024 * 1024  # 100 MB
# --- The maximum number of tokens to extract from any single text/PDF file.
MAX_FILE_TOKENS = 500_000
# --- The maximum size for any single uploaded file.
MAX_FILE_SIZE_BYTES = 50 * 1024 * 1024  # 50 MB
# --- The target compressed size for images to optimize token usage.
TARGET_IMAGE_SIZE_BYTES = 2000 * 1024  # 2 MB
# --- Maximum size for any binary file to be inlined directly into a prompt.
MAX_BINARY_INLINE_BYTES = 10 * 1024 * 1024 # 10 MB

# -----------------------------------------------------------------------------
# --- Pydantic Models for API Requests
# -----------------------------------------------------------------------------
# These models define the expected structure and data types for the API endpoints,
# ensuring data validation and clear API contracts.
# -----------------------------------------------------------------------------

class CachedFile(BaseModel):
    """
    Represents a file that is already cached (e.g., in S3) and can be
    referenced by its URI instead of being re-uploaded.
    """
    uri: str = Field(alias="URI", description="The URI of the cached file (e.g., 's3://bucket/key').")
    file_name: str = Field(alias="FileName", description="The original name of the file.")
    mime_type: str = Field(alias="MimeType", description="The MIME type of the file.")

class LLMSettings(BaseModel):
    """
    Defines settings for using a custom, external LLM provider for a specific request,
    overriding the application's default provider.
    """
    provider: str = Field(description="The name of the LLM provider (e.g., 'openai', 'anthropic').")
    api_key: str = Field(description="The API key for the specified provider.")
    model: str = Field(description="The specific model to use from the provider (e.g., 'gpt-4o').")

class EgoRequest(BaseModel):
    """
    Represents the main request body for interacting with the EGO agent's
    thought generation and synthesis endpoints.
    """
    query: str = Field(description="The user's primary query or message to the agent.")
    mode: str = Field(description="The operational mode for the agent (e.g., 'agent', 'research', 'default').")
    chat_history: str = Field(default="", description="The history of the conversation so far.")
    thoughts_history: str = Field(default="", description="The history of the agent's previous internal thoughts.")
    custom_instructions: Optional[str] = Field(None, description="User-provided custom instructions to guide the agent's behavior.")
    files: List[Any] = Field(default=[], description="A list of files being uploaded with the request (handled by FastAPI).")
    cached_files: List[CachedFile] = Field(default=[], description="A list of already cached files to be used.")
    retrieved_snippets: List[str] = Field(default_factory=list, description="Context snippets retrieved from an external vector search.")
    user_id: Optional[str] = Field(None, description="The unique identifier for the user.")
    session_id: Optional[int] = Field(None, description="[Legacy] The unique ID of the chat session.")
    session_uuid: Optional[str] = Field(None, description="The unique UUID of the chat session.")
    log_id: Optional[int] = Field(None, description="The ID of the log entry for this specific interaction, used for memory association.")
    regenerated_log_id: Optional[int] = Field(None, description="If regenerating a response, the ID of the log to start deleting memory from.")
    llm_settings: Optional[LLMSettings] = Field(None, description="Custom LLM provider settings for this request.")
    memory_enabled: Optional[bool] = Field(True, description="Whether the agent's long-term memory is enabled for this request.")
    session_created_at: Optional[str] = Field(None, description="The ISO 8601 timestamp of when the session was created.")

class ToolExecutionRequest(BaseModel):
    """Request model for the endpoint that executes a specific tool."""
    query: str = Field(description="The input query, command, or data for the tool.")
    user_id: Optional[str] = Field(None, description="The ID of the user, required for tools like EgoMemory.")
    memory_enabled: bool = Field(True, description="Flag indicating if memory can be used by the tool.")

class LLMSettingsRequest(BaseModel):
    """Request model for validating an LLM provider's API key."""
    provider: str = Field(description="The name of the LLM provider to validate.")
    api_key: str = Field(description="The API key to be validated.")

class ModelListRequest(BaseModel):
    """Request model for listing available models from an LLM provider."""
    provider: str = Field(description="The name of the LLM provider.")
    api_key: str = Field(description="The API key to use for authentication.")

class TitleRequest(BaseModel):
    """Request model for the chat title generation endpoint."""
    text: str = Field(description="The initial text of the chat to be summarized into a title.")

# -----------------------------------------------------------------------------
# --- Application Lifecycle (Lifespan)
# -----------------------------------------------------------------------------

@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Asynchronous context manager to handle application startup and shutdown events.

    During startup, it initializes a reusable S3 session and configuration, attaching
    them to the application's state for access in other parts of the code. During
    shutdown, it ensures resources are gracefully released.

    Args:
        app: The FastAPI application instance.
    """
    log.info("Application startup: Initializing S3 session...")
    try:
        # --- Initialize the S3 session object once for the entire application lifecycle.
        app.state.s3_session = aioboto3.Session()
        
        # --- Read S3 connection parameters from environment variables.
        endpoint = os.getenv("S3_ENDPOINT")
        if endpoint and not endpoint.startswith(("http://", "https://")):
            endpoint = f"https://{endpoint.strip()}"

        # Build s3_params, but only include explicit credentials if provided.
        # This allows AWS default credential provider chain (e.g., IAM role) to work in deployment.
        s3_params: dict = {
            "config": Config(s3={"addressing_style": "path"})  # Required for some S3-compatible services like MinIO
        }
        if endpoint:
            s3_params["endpoint_url"] = endpoint

        region = os.getenv("S3_REGION") or os.getenv("AWS_REGION") or os.getenv("AWS_DEFAULT_REGION")
        if region:
            s3_params["region_name"] = region

        # Accept both ID-style and short-style env var names for compatibility
        access_key = os.getenv("S3_ACCESS_KEY_ID") or os.getenv("S3_ACCESS_KEY")
        secret_key = os.getenv("S3_SECRET_ACCESS_KEY") or os.getenv("S3_SECRET_KEY")
        if access_key and secret_key:
            s3_params["aws_access_key_id"] = access_key
            s3_params["aws_secret_access_key"] = secret_key
        elif access_key or secret_key:
            log.warning("Partial S3 credentials provided: access_key=%s secret_key=%s. Falling back to default provider chain.",
                        'yes' if access_key else 'no', 'yes' if secret_key else 'no')

        app.state.s3_params = s3_params
        app.state.S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME")

        # Log what will be used (without secrets)
        log.info(
            "S3 configured | endpoint=%s region=%s explicit_keys=%s",
            s3_params.get("endpoint_url", "<default>"),
            s3_params.get("region_name", "<default>"),
            "yes" if ("aws_access_key_id" in s3_params and "aws_secret_access_key" in s3_params) else "no (using default provider chain)"
        )
        log.info("S3 session and configuration initialized successfully.")
    except Exception as e:
        log.error(f"Failed to initialize S3 session during startup: {e}", exc_info=True)
        # --- Set state to None to handle failures gracefully in endpoints.
        app.state.s3_session = None
        app.state.s3_params = {}

    # --- The `yield` statement passes control to the running application.
    yield
    
    # --- Code after the `yield` is executed during application shutdown.
    log.info("Application shutdown: Releasing resources.")
    # --- Here you would add cleanup code, e.g., closing the S3 session if the library required it.
    # --- `aioboto3` session management doesn't require explicit closing.

# -----------------------------------------------------------------------------
# --- FastAPI App and Core EGO Initialization
# -----------------------------------------------------------------------------

try:
    # --- Create the main FastAPI application instance with the lifespan manager.
    app = FastAPI(lifespan=lifespan)
    
    # --- Configure CORS (Cross-Origin Resource Sharing) middleware.
    app.add_middleware(
        CORSMiddleware,
        allow_origins=[os.getenv("CORS_ALLOW_ORIGINS", "*")], # Be more specific in production
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # --- Initialize the default, internal LLM backend (EgoGeminiProvider).
    default_backend = EgoGeminiProvider()
    
    # --- Initialize Vector Memory if a database URL is configured.
    vector_memory: Optional[VectorMemory] = None
    if os.getenv("DATABASE_URL") or os.getenv("EGO_MEMORY_DB_URL"):
        try:
            vector_memory = VectorMemory(backend=default_backend)
            log.info("VectorMemory (pgvector) has been successfully initialized.")
        except Exception as e:
            # --- If memory fails to initialize, log the error but allow the app to start without it.
            log.error(f"Failed to initialize VectorMemory. The agent will run without long-term memory. Error: {e}", exc_info=True)
            vector_memory = None
    else:
        log.warning("VectorMemory is disabled: DATABASE_URL environment variable is not set.")

    # --- Assemble the list of tools available to the EGO agent.
    tools = [
        EgoSearch(backend=default_backend),
        AlterEgo(backend=default_backend),
        EgoCalc(),
        EgoWiki(),
        EgoTube(backend=default_backend),
    ]
    # --- Conditionally enable the code execution tool based on an environment variable.
    if os.getenv("EGO_ENABLE_CODEEXEC", "0").lower() in ("1", "true", "yes"):
        tools.append(EgoCodeExec(backend=default_backend))
    # --- Only add the memory tool if the vector memory was successfully initialized.
    if vector_memory:
        tools.append(EgoMemory(vector_memory=vector_memory))

    # --- Initialize the main EGO agent instance with the backend and tools.
    ego_instance = EGO(backend=default_backend, tools=tools)
    log.info(f"EGO agent initialized successfully with {len(tools)} tools.")

except Exception as e:
    # --- A critical error during initialization is fatal. The application cannot run.
    log.critical(f"A critical error occurred during application initialization: {e}", exc_info=True)
    # --- In a real deployment, this would cause the process to exit and be restarted by a manager.

# -----------------------------------------------------------------------------
# --- File Processing Helper Functions
# -----------------------------------------------------------------------------
# These async functions handle the processing of different file types,
# preparing them to be consumed by the LLM.
# -----------------------------------------------------------------------------

async def _process_image(tmp_file, name: str) -> Optional[Part]:
    """
    Compresses and prepares an image file to be sent to the model as a Google GenAI `Part`.

    This function attempts to convert the image to JPEG and iteratively reduces its
    quality and dimensions to meet the `TARGET_IMAGE_SIZE_BYTES` constant, optimizing
    for token cost and performance.

    Args:
        tmp_file: A file-like object containing the image data.
        name: The filename of the image, used for logging.

    Returns:
        A `google.genai.types.Part` object containing the compressed image data,
        or `None` if an error occurs.
    """
    print(f"[DEBUG] _process_image called for: {name}")
    try:
        # --- Ensure we're at the beginning of the file
        tmp_file.seek(0)
        # --- Open the image using Pillow.
        with Image.open(tmp_file) as img:
            log.debug(f"Processing image: {name}, original size: {img.size}, mode: {img.mode}")
            # --- Ensure image is in a standard format (RGB) for compatibility.
            if img.mode in ('P', 'RGBA'):
                img = img.convert('RGB')

            # --- Iteratively compress the image to meet the size target.
            quality = 85
            max_side = max(img.size)
            for _ in range(6):  # Limit attempts to avoid infinite loops.
                buf = io.BytesIO()
                img.save(buf, format='JPEG', quality=quality, optimize=True)
                data = buf.getvalue()
                # --- If the image is small enough, we're done.
                if len(data) <= TARGET_IMAGE_SIZE_BYTES or (quality <= 50 and max_side <= 1024):
                    log.debug(f"Image {name} compressed to {len(data)} bytes.")
                    print(f"[DEBUG] _process_image - successfully compressed {name} to {len(data)} bytes")
                    return Part.from_bytes(data=data, mime_type='image/jpeg')

                # --- If still too large, reduce quality and resize for the next attempt.
                quality = max(50, quality - 10)
                if max_side > 1024:
                    scale = 1024 / max_side
                    img = img.resize((int(img.width * scale), int(img.height * scale)), Image.Resampling.LANCZOS)
                    max_side = max(img.size)
            
            # --- If all attempts fail, use the last compressed version.
            log.warning(f"Could not compress image '{name}' to target size. Using last attempt of {len(data)} bytes.")
            return Part.from_bytes(data=data, mime_type='image/jpeg')
            
    except Exception as e:
        log.error(f"An exception occurred while processing image '{name}': {e}", exc_info=True)
        return None

async def _process_pdf(tmp_path: str, name: str) -> str:
    """
    Extracts text content from a PDF file using PyMuPDF (fitz).

    It reads the text from each page and concatenates it, stopping early if the
    extracted text exceeds the `MAX_FILE_TOKENS` limit to prevent excessive prompt length.

    Args:
        tmp_path: The local filesystem path to the temporary PDF file.
        name: The filename of the PDF, used for context and logging.

    Returns:
        A formatted string containing the extracted text, or an error message.
    """
    try:
        # --- Using a context manager ensures the document is properly closed.
        with fitz.open(tmp_path) as doc:
            texts: List[str] = []
            total_len = 0
            for page in doc:
                page_text = page.get_text("text", sort=True).strip()
                if page_text:
                    texts.append(page_text)
                    total_len += len(page_text)
                    # --- Stop processing if the file is excessively large to save resources.
                    if total_len > MAX_FILE_TOKENS * 4: # Using 4 as a heuristic for chars/token
                        log.warning(f"Stopped processing PDF '{name}' early due to token limit.")
                        break
            
            full_text = "\n\n".join(texts)
            # --- Return a specially formatted block for the LLM.
            return f"[File: {name}]\n{full_text[:MAX_FILE_TOKENS * 4]}"
    except Exception as e:
        log.error(f"Error processing PDF file '{name}': {e}", exc_info=True)
        return f"[File: {name} - Error processing PDF: {str(e)}]"

async def _process_text(tmp_file, name: str) -> str:
    """
    Extracts text content from a plain text file.

    It reads the file up to the token limit and decodes it as UTF-8.

    Args:
        tmp_file: A file-like object for the text file.
        name: The filename, used for context.

    Returns:
        A formatted string with the file's content, or an error message.
    """
    try:
        # --- Read up to the character limit and decode safely.
        data = tmp_file.read(MAX_FILE_TOKENS * 4)
        text = data.decode('utf-8', errors='replace')
        return f"[File: {name}]\n{text}"
    except Exception as e:
        log.error(f"Error processing text file '{name}': {e}", exc_info=True)
        return f"[File: {name} - Error reading file]"

async def process_files(request: Request, files: List[UploadFile], cached_files: List[CachedFile]) -> List[Any]:
    """
    Processes all uploaded and cached files for a request, preparing them for the LLM.

    This function orchestrates the processing of different file types, respects size
    and token limits, and intelligently decides whether to inline content or use the
    Gemini File API for large files. It handles both direct uploads and references
    to files stored in S3.

    Args:
        request: The incoming FastAPI request, used to access app state (e.g., S3 client).
        files: A list of files uploaded with the current request.
        cached_files: A list of `CachedFile` objects referencing files already in S3.

    Returns:
        A list of parts (text, dictionaries, or `google.genai.types.Part` objects)
        to be included in the LLM prompt.
    """
    print(f"[DEBUG] process_files called with {len(files)} uploaded files and {len(cached_files)} cached files")
    parts: List[Any] = []
    combined_bytes = 0

    # --- Step 1: Handle newly uploaded files.
    for up in files or []:
        mime = up.content_type or 'application/octet-stream'
        name = up.filename or 'file'

        # --- Use a temporary file on disk to handle large uploads without consuming all RAM.
        with NamedTemporaryFile(mode='w+b', delete=True) as tmp:
            file_bytes, too_large = 0, False
            try:
                # --- Read the uploaded file in chunks into the temporary file.
                while chunk := await up.read(64 * 1024):
                    file_bytes += len(chunk)
                    if file_bytes > MAX_FILE_SIZE_BYTES or (combined_bytes + file_bytes) > MAX_COMBINED_FILE_BYTES:
                        too_large = True
                        break
                    tmp.write(chunk)
                combined_bytes += file_bytes
            except Exception as e:
                log.error(f"Error reading uploaded file chunk for {name}: {e}")
                continue

            if too_large:
                parts.append(f"[File: {name} - Skipped: size exceeds limits.]")
                if combined_bytes > MAX_COMBINED_FILE_BYTES:
                    log.warning("Combined file size limit reached, stopping further file processing.")
                    break
                continue

            tmp.seek(0) # Rewind the temp file to be read by processors.
            
            # --- Route to the correct processor based on MIME type.
            if mime.startswith('image/'):
                img_part = await _process_image(tmp, name)
                if img_part: parts.append(img_part)
                else: parts.append(f"[Image: {name} - Failed to process]")
            elif mime in ['application/pdf', 'text/plain', 'text/markdown', 'text/csv']:
                 if mime == 'application/pdf':
                    content = await _process_pdf(tmp.name, name)
                    parts.append({"type": "file", "name": name, "content": content})
                 else:
                    content = await _process_text(tmp, name)
                    parts.append({"type": "file", "name": name, "content": content})
            else:
                # --- For other file types, just pass a reference to be used by tools.
                parts.append(f"[File available for tools: {name} ({mime})]")

    # --- Step 2: Handle cached files from S3.
    print(f"[DEBUG] process_files - processing {len(cached_files)} cached files")
    for i, cf in enumerate(cached_files or []):
        mime = cf.mime_type or 'application/octet-stream'
        name = cf.file_name or 'cached-file'
        uri = cf.uri or ''
        print(f"[DEBUG] process_files - cached file {i+1}: name='{name}', uri='{uri}', mime='{mime}'")
        
        # --- Attempt to download supported types (images, pdf, text) from S3 and inline them.
        if uri.startswith('s3://') and request.app and getattr(request.app.state, 's3_session', None):
            try:
                # --- Parse s3://bucket/key URI format.
                _, rest = uri.split('://', 1)
                bucket, key = rest.split('/', 1)

                # Respect combined and per-file limits.
                max_bytes_to_read = min(MAX_FILE_SIZE_BYTES, MAX_COMBINED_FILE_BYTES - combined_bytes)
                if max_bytes_to_read <= 0:
                    parts.append(f"[Cached file skipped due to combined size limit: {name}]")
                    continue

                async with request.app.state.s3_session.client('s3', **request.app.state.s3_params) as s3:
                    obj = await s3.get_object(Bucket=bucket, Key=key)
                    body = obj['Body']
                    with NamedTemporaryFile(mode='w+b', delete=True) as tmp:
                        read_bytes = 0
                        while chunk := await body.read(64 * 1024):
                            if (read_bytes + len(chunk)) > max_bytes_to_read:
                                tmp.write(chunk[:max_bytes_to_read - read_bytes])
                                read_bytes = max_bytes_to_read
                                break
                            tmp.write(chunk)
                            read_bytes += len(chunk)
                        combined_bytes += read_bytes
                        tmp.seek(0)

                        if mime.startswith('image/'):
                            print(f"[DEBUG] process_files - processing cached image: {name}")
                            img_part = await _process_image(tmp, name)
                            if img_part: 
                                print(f"[DEBUG] process_files - successfully processed cached image: {name}")
                                parts.append(img_part)
                            else: 
                                print(f"[DEBUG] process_files - failed to process cached image: {name}")
                                parts.append(f"[Image: {name} - Failed to process]")
                        elif mime == 'application/pdf':
                            content = await _process_pdf(tmp.name, name)
                            parts.append({"type": "file", "name": name, "content": content})
                        elif mime in ['text/plain', 'text/markdown', 'text/csv']:
                            content = await _process_text(tmp, name)
                            parts.append({"type": "file", "name": name, "content": content})
                        else:
                            parts.append(f"[Cached file available for tools: {name} at {cf.uri} ({mime})]")
            except Exception as e:
                log.error(f"Failed to process cached file {name} from {uri}: {e}", exc_info=True)
                parts.append(f"[Cached file available for tools (download failed): {name} at {uri}]")
        else:
            # --- For non-S3 or unsupported cases, just pass a reference to be used by tools.
            parts.append(f"[Cached file available for tools: {name} at {cf.uri} ({mime})]")

    return parts


# -----------------------------------------------------------------------------
# --- API Endpoints
# -----------------------------------------------------------------------------

@app.get("/healthcheck", summary="Health Check")
async def health_check():
    """A simple endpoint to confirm that the service is running and responsive."""
    return JSONResponse(content={"status": "ok"})

@app.post("/generate_thought")
async def generate_thought(request: Request):
    request_data = await request.form()
    ego_req = EgoRequest.parse_raw(request_data["request_data"])
    
    # DEBUG: Log chat_history details
    print(f"[DEBUG] generate_thought - chat_history length: {len(ego_req.chat_history)}")
    print(f"[DEBUG] generate_thought - chat_history preview: {ego_req.chat_history[:200]}...")
    print(f"[DEBUG] generate_thought - session_uuid: {ego_req.session_uuid}")
    print(f"[DEBUG] generate_thought - thoughts_history length: {len(ego_req.thoughts_history)}")
    print(f"[DEBUG] generate_thought - thoughts_history preview: {ego_req.thoughts_history[:200]}...")
    
    files_data = []
    print(f"[DEBUG] generate_thought - request_data keys: {list(request_data.keys())}")
    for key, file in request_data.items():
        if key != "request_data" and hasattr(file, 'read'):
            content = await file.read()
            print(f"[DEBUG] generate_thought - found file: {file.filename}, content_type: {file.content_type}, size: {len(content)}")
            files_data.append(file)  # Pass the actual UploadFile object
    print(f"[DEBUG] generate_thought - total files_data: {len(files_data)}")
    
    try:
        # --- Dynamically select the LLM provider for this request.
        backend = get_llm_provider(ego_req.llm_settings.provider, ego_req.llm_settings.api_key) if ego_req.llm_settings else default_backend
        
        prompt_parts = await process_files(request, files_data, ego_req.cached_files)
        log.debug(f"Processed {len(prompt_parts)} file parts for thought generation.")

        # --- The agent's generate_thought method encapsulates the core logic.
        thought_json, usage = await ego_instance.generate_thought(
            query=ego_req.query, mode=ego_req.mode, chat_history=ego_req.chat_history,
            thoughts_history=ego_req.thoughts_history, custom_instructions=ego_req.custom_instructions,
            prompt_parts_from_files=prompt_parts, model=ego_req.llm_settings.model if ego_req.llm_settings else None,
            vector_memory=vector_memory, user_id=ego_req.user_id, session_uuid=ego_req.session_uuid,
            current_log_id=ego_req.log_id, memory_enabled=ego_req.memory_enabled
        )
        
        # Execute tool calls if present
        tool_results = []
        if thought_json and isinstance(thought_json, dict) and thought_json.get('tool_calls'):
            tool_calls = thought_json.get('tool_calls', [])
            log.info(f"Executing {len(tool_calls)} tool calls from thought")
            
            for i, tool_call in enumerate(tool_calls, 1):
                tool_name = tool_call.get('tool_name')
                tool_query = tool_call.get('tool_query')
                
                if not tool_name or not tool_query:
                    continue
                
                # Send progress update for current tool
                if len(tool_calls) > 1:
                    progress_header = f"Using {tool_name}... ({i}/{len(tool_calls)})"
                else:
                    progress_header = f"Using {tool_name}..."
                
                # Add progress update to tool_results for immediate frontend feedback
                tool_results.append({
                    "type": "tool_progress",
                    "tool_name": tool_name,
                    "progress": f"{i}/{len(tool_calls)}",
                    "header": progress_header
                })
                    
                try:
                    # Get the tool from ego_instance
                    tool = ego_instance.tools.get(tool_name)
                    if not tool:
                        result = f"Tool '{tool_name}' not found or is unavailable."
                    elif tool_name == "EgoMemory" and not ego_req.memory_enabled:
                        result = "Error: Memory is disabled in the session settings."
                    else:
                        result = await tool.use(tool_query, user_id=ego_req.user_id)
                        result = str(result)
                    
                    # Send completion update
                    if len(tool_calls) > 1:
                        completion_header = f"{tool_name} completed ({i}/{len(tool_calls)})"
                    else:
                        completion_header = f"{tool_name} completed"
                    
                    tool_results.append({
                        "type": "tool_progress",
                        "tool_name": tool_name,
                        "progress": f"{i}/{len(tool_calls)}",
                        "header": completion_header
                    })
                    
                    tool_results.append({
                        "type": "tool_output",
                        "tool_name": tool_name,
                        "output": result
                    })
                    log.info(f"Tool '{tool_name}' executed successfully")
                    
                except Exception as e:
                    log.error(f"Error executing tool '{tool_name}': {e}", exc_info=True)
                    
                    # Send error update
                    error_header = f"{tool_name} failed ({i}/{len(tool_calls)})"
                    tool_results.append({
                        "type": "tool_progress",
                        "tool_name": tool_name,
                        "progress": f"{i}/{len(tool_calls)}",
                        "header": error_header
                    })
                    
                    tool_results.append({
                        "type": "tool_error", 
                        "tool_name": tool_name,
                        "error": str(e)
                    })

        # Create formatted thought_entry for frontend accumulation
        thought_entry = ""
        if thought_json and isinstance(thought_json, dict):
            header = thought_json.get('thoughts_header', 'Thinking')
            content = thought_json.get('thoughts', '')
            if content:
                thought_entry = f"## {header}\n{content}"
        
        return JSONResponse(content={
            "thought": thought_json, 
            "thought_entry": thought_entry, 
            "tool_results": tool_results,
            "usage": usage
        })
    except Exception as e:
        log.error(f"Error in generate_thought endpoint: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail="An internal server error occurred during thought generation.")

@app.post("/synthesize_stream", summary="Generate Final Response Stream")
async def synthesize_stream(request: Request, request_data: str = Form(...), files: List[UploadFile] = File(default=[])):
    """
    Generates the agent's final, user-facing response as a server-sent event (SSE) stream.
    """
    try:
        ego_req = EgoRequest.parse_raw(request_data)
        
        # DEBUG: Log thoughts_history details for synthesis
        print(f"[DEBUG] synthesize_stream - thoughts_history length: {len(ego_req.thoughts_history)}")
        print(f"[DEBUG] synthesize_stream - thoughts_history preview: {ego_req.thoughts_history[:300]}...")
        
        backend = get_llm_provider(ego_req.llm_settings.provider, ego_req.llm_settings.api_key) if ego_req.llm_settings else default_backend
        
        prompt_parts = await process_files(request, files, ego_req.cached_files)
        log.debug(f"Processed {len(prompt_parts)} file parts for synthesis stream.")

        async def event_generator() -> AsyncGenerator[str, None]:
            """This inner function is the actual async generator for the SSE stream."""
            try:
                # --- Handle memory cleanup if this is a response regeneration.
                if vector_memory and ego_req.regenerated_log_id and ego_req.user_id and ego_req.session_uuid:
                    await vector_memory.delete_at_or_after(ego_req.user_id, ego_req.session_uuid, int(ego_req.regenerated_log_id))

                # --- Get the stream from the EGO agent instance.
                stream = ego_instance.synthesize_stream(
                    query=ego_req.query, mode=ego_req.mode, chat_history=ego_req.chat_history,
                    thoughts_history=ego_req.thoughts_history, custom_instructions=ego_req.custom_instructions,
                    prompt_parts_from_files=prompt_parts, model=ego_req.llm_settings.model if ego_req.llm_settings else None,
                    vector_memory=vector_memory, user_id=ego_req.user_id, session_uuid=ego_req.session_uuid,
                    current_log_id=ego_req.log_id, memory_enabled=ego_req.memory_enabled
                )
                
                full_response_text = []
                async for chunk in stream:
                    # --- Format each chunk as a server-sent event.
                    sse_event = {"type": "chunk", "data": {"text": chunk}}
                    yield f"data: {json.dumps(sse_event, ensure_ascii=False)}\n\n"
                    full_response_text.append(chunk)

                # --- After the stream is complete, add the final conversation turn to memory.
                if vector_memory and ego_req.user_id and ego_req.memory_enabled:
                    final_text = f"User: {ego_req.query}\nEGO: {''.join(full_response_text)}"
                    sess_id = ego_req.session_uuid or str(ego_req.session_id or '')
                    await vector_memory.add_texts(
                        user_id=ego_req.user_id, session_id=sess_id, texts=[final_text],
                        log_id=ego_req.log_id, session_created_at=ego_req.session_created_at
                    )
            except Exception as e_gen:
                log.error(f"Error inside SSE event generator: {e_gen}", exc_info=True)
                err_event = {"type": "error", "data": {"message": "An error occurred while generating the response."}}
                yield f"data: {json.dumps(err_event, ensure_ascii=False)}\n\n"

        # --- Return a StreamingResponse that uses the event generator.
        return StreamingResponse(event_generator(), media_type="text/event-stream")
    except Exception as e:
        log.error(f"Error processing synthesize_stream request: {e}", exc_info=True)
        raise HTTPException(status_code=400, detail="Error processing the incoming request.")

@app.post("/execute_tool/{tool_name}", summary="Execute a Specific Tool")
async def execute_tool(tool_name: str, request: ToolExecutionRequest):
    """
    Executes a specified tool with the given query and returns the result.
    This allows the frontend to call tools directly if needed.
    """
    try:
        tool = ego_instance.tools.get(tool_name)
        if not tool:
            return JSONResponse(content={"result": f"Tool '{tool_name}' not found or is unavailable."})

        # --- Special check for the memory tool.
        if tool_name == "EgoMemory" and not request.memory_enabled:
            return JSONResponse(content={"result": "Error: Memory is disabled in the session settings."})

        result = await tool.use(request.query, user_id=request.user_id)
        return JSONResponse(content={"result": str(result)})
    except Exception as e:
        log.error(f"Error executing tool '{tool_name}': {e}", exc_info=True)
        # --- Return a generic error to the user in a JSON format.
        return JSONResponse(content={"result": "The tool failed to execute the request. Please try again later."})

@app.post("/generate_title", summary="Generate Chat Title")
async def generate_title(req: TitleRequest):
    """
    Generates a concise title for a chat based on its initial text,
    using the default backend's title generation capability.
    """
    try:
        # --- This assumes the default backend has a `generate_title` method.
        if hasattr(default_backend, 'generate_title'):
             title = await default_backend.generate_title(req.text)
             return JSONResponse(content={"title": title})
        else:
             log.warning("The default backend does not support title generation.")
             return JSONResponse(content={"title": "New Chat"})
    except Exception as e:
        log.error(f"Error generating title: {e}", exc_info=True)
        # --- Provide a safe fallback title on error.
        return JSONResponse(content={"title": "New Chat"})
# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import logging
import os
import sys

# --- External dependency for remote logging to BetterStack.
# --- This is wrapped in a try-except block to prevent the application from
# --- crashing if the library is not installed, especially if it's an optional feature.
try:
    from logtail import LogtailHandler
except ImportError:
    # --- If the library is missing, create a dummy class so the rest of the code
    # --- can run without errors, effectively disabling the feature.
    LogtailHandler = None
    # --- A flag to check if the import was successful.
    LOGTAIL_AVAILABLE = False
else:
    LOGTAIL_AVAILABLE = True

# -----------------------------------------------------------------------------
# --- Core Functions
# -----------------------------------------------------------------------------

def setup_logging():
    """
    Configures and sets up the root logger for the entire application.

    This function should be called once at the very beginning of the application's
    lifecycle. It reads environment variables to determine the logging level and
    where the logs should be sent (console, BetterStack, or both). It clears any
    pre-existing handlers to ensure a clean configuration.

    Configuration is controlled by the following environment variables:
        - LOG_LEVEL: The minimum level of logs to capture (e.g., 'DEBUG', 'INFO',
          'WARNING', 'ERROR'). Defaults to 'INFO'.
        - LOGTAIL_SOURCE_TOKEN: Your source token from BetterStack. If this is set,
          logs will be sent to your BetterStack account.
        - LOG_TO_CONSOLE: Can be set to 'false' to disable logging to the console
          (stdout). Defaults to 'true'.
    """
    # --- Determine the logging level from environment variables, with a safe default.
    log_level_name = os.getenv("LOG_LEVEL", "INFO").upper()
    log_level = getattr(logging, log_level_name, logging.INFO)

    # --- Define a standard format for all log messages.
    log_format = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s (%(filename)s:%(lineno)d)'
    )

    # --- Get the root logger, which is the top-level logger for the application.
    root_logger = logging.getLogger()
    root_logger.setLevel(log_level)

    # --- It's good practice to clear any existing handlers that might have been
    # --- configured by other libraries or previous setups, to avoid duplicate logs.
    if root_logger.hasHandlers():
        root_logger.handlers.clear()

    # --- Configuration for the Logtail (BetterStack) handler.
    # --- This handler sends logs to an external service.
    source_token = os.getenv("LOGTAIL_SOURCE_TOKEN")
    if source_token:
        if LOGTAIL_AVAILABLE:
            try:
                logtail_handler = LogtailHandler(source_token=source_token)
                logtail_handler.setFormatter(log_format)
                root_logger.addHandler(logtail_handler)
                # --- Use logging.info for startup messages for consistency.
                logging.info("Logging to BetterStack is enabled.")
            except Exception as e:
                # --- This handles potential errors during LogtailHandler initialization.
                logging.error(f"Failed to initialize BetterStack logging: {e}", exc_info=True)
        else:
            # --- Inform the user if the token is set but the library is missing.
            logging.warning("LOGTAIL_SOURCE_TOKEN is set, but the 'logtail-handler' library is not installed. Remote logging is disabled.")
    else:
        logging.info("Logging to BetterStack is disabled (LOGTAIL_SOURCE_TOKEN not set).")

    # --- Configuration for the console handler.
    # --- This handler prints logs to the standard output (your terminal).
    if os.getenv("LOG_TO_CONSOLE", "true").lower() == "true":
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setFormatter(log_format)
        root_logger.addHandler(console_handler)
        logging.info("Logging to console is enabled.")
    
    # --- If no handlers were configured at all, add a basic console logger
    # --- to ensure that critical messages are not lost.
    if not root_logger.hasHandlers():
        logging.basicConfig(level=log_level)
        logging.warning("No log handlers were configured. Falling back to basic console logging.")


def get_logger(name: str) -> logging.Logger:
    """
    Returns a logger instance for a specific module.

    This is a convenience wrapper around `logging.getLogger()`. Using this function
    ensures a consistent way of obtaining logger instances throughout the application.
    It's best practice to create a logger for each module.

    Example Usage:
        `log = get_logger(__name__)`
        `log.info("This is a log message from my module.")`

    Args:
        name: The name for the logger. This should almost always be `__name__`,
              which automatically uses the module's name.

    Returns:
        A `logging.Logger` instance that is a child of the root logger.
    """
    return logging.getLogger(name)
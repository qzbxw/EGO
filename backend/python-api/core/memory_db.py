# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import os
import asyncio
import logging
from datetime import datetime
from typing import List, Any, Optional

import asyncpg
from pydantic import BaseModel

# --- Local Module Imports
from .llm_backend import LLMProvider

# -----------------------------------------------------------------------------
# --- Logging Configuration
# -----------------------------------------------------------------------------
log = logging.getLogger(__name__)

# -----------------------------------------------------------------------------
# --- Data Models
# -----------------------------------------------------------------------------

class MemoryHit(BaseModel):
    """
    Represents a single memory item retrieved from the vector database.
    """
    user_id: str
    session_id: str
    text: str
    score: float
    created_at: Optional[str] = None
    session_created_at: Optional[str] = None
    log_id: Optional[int] = None

# -----------------------------------------------------------------------------
# --- Core Class: VectorMemory
# -----------------------------------------------------------------------------

class VectorMemory:
    """
    Manages the agent's long-term memory using a PostgreSQL vector database.
    """

    def __init__(self, backend: LLMProvider, db_url: Optional[str] = None, max_entries_per_user: int = 5000):
        """
        Initializes the VectorMemory instance.
        """
        self.backend = backend
        self.db_url = db_url or os.getenv("DATABASE_URL") or os.getenv("EGO_MEMORY_DB_URL")
        if not self.db_url:
            log.critical("DATABASE_URL or EGO_MEMORY_DB_URL is required for VectorMemory but was not found.")
            raise ValueError("A database URL is required for VectorMemory.")

        if "sslmode" not in self.db_url and self.db_url.startswith(("postgres://", "postgresql://")):
            separator = "&" if "?" in self.db_url else "?"
            self.db_url = f"{self.db_url}{separator}sslmode=require"

        self._pool: Optional[asyncpg.Pool] = None
        self._init_lock = asyncio.Lock()
        self._max_entries = max_entries_per_user
        self._dim = 256

    async def _init(self):
        """
        Initializes the database connection pool and ensures the schema exists.
        """
        if self._pool:
            return
        async with self._init_lock:
            if self._pool:
                return
            try:
                log.info("Initializing VectorMemory database connection pool...")
                self._pool = await asyncpg.create_pool(self.db_url, min_size=1, max_size=5)
                
                async with self._pool.acquire() as conn:
                    log.info("Ensuring 'vector' extension and 'ego_memory' table schema exist.")
                    await conn.execute("CREATE EXTENSION IF NOT EXISTS vector")
                    await conn.execute(
                        f"""
                        CREATE TABLE IF NOT EXISTS ego_memory (
                            id BIGSERIAL PRIMARY KEY,
                            user_id TEXT NOT NULL,
                            session_id TEXT NOT NULL,
                            text TEXT NOT NULL,
                            vector VECTOR({self._dim}) NOT NULL,
                            created_at TIMESTAMPTZ DEFAULT NOW(),
                            session_created_at TIMESTAMPTZ DEFAULT NOW(),
                            log_id BIGINT NULL UNIQUE
                        )
                        """
                    )
                    await conn.execute("CREATE INDEX IF NOT EXISTS ego_memory_user_idx ON ego_memory (user_id, created_at DESC)")
                    await conn.execute("CREATE INDEX IF NOT EXISTS ego_memory_log_id_idx ON ego_memory (log_id)")
                    
                    try:
                        await conn.execute(
                            "CREATE INDEX IF NOT EXISTS ego_memory_vec_cos_ivfflat ON ego_memory USING ivfflat (vector vector_cosine_ops) WITH (lists=100)"
                        )
                    except asyncpg.PostgresError as e:
                        log.warning(f"Could not create IVFFlat vector index (this is non-critical): {e}")

                log.info("VectorMemory database initialization complete.")
            except (asyncpg.PostgresError, OSError) as e:
                log.critical(f"Failed to initialize VectorMemory database connection: {e}", exc_info=True)
                self._pool = None
                raise RuntimeError(f"Database initialization failed: {e}")

    def _to_vector_literal(self, vec: List[float]) -> str:
        """Converts a list of floats into a string literal for pgvector."""
        return "[" + ",".join(map(str, vec)) + "]"

    async def add_texts(self, user_id: str, session_id: str, texts: List[str], log_id: Optional[int] = None, session_created_at: Optional[str] = None) -> None:
        """
        Embeds and adds a list of texts to the user's memory, then prunes old entries.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot add texts; database pool is not initialized.")
            return

        rows: List[List[Any]] = []
        session_dt = None
        if session_created_at:
            try:
                session_dt = datetime.fromisoformat(session_created_at.replace('Z', '+00:00'))
            except (ValueError, TypeError) as e:
                log.warning(f"Failed to parse session_created_at '{session_created_at}': {e}")

        for text in texts:
            if not text or not text.strip():
                continue
            try:
                vec = await self.backend.embed(text)
                if len(vec) != self._dim:
                    vec = (vec + [0.0] * self._dim)[:self._dim]
                rows.append([user_id, str(session_id), text, self._to_vector_literal(vec), log_id, session_dt])
            except Exception as e:
                log.error(f"Failed to embed text for memory. Text: '{text[:50]}...'. Error: {e}", exc_info=True)
        
        if not rows:
            log.info(f"No valid texts to add to memory for user '{user_id}'.")
            return

        try:
            async with self._pool.acquire() as conn:
                async with conn.transaction():
                    await conn.executemany(
                        "INSERT INTO ego_memory (user_id, session_id, text, vector, log_id, session_created_at) VALUES ($1, $2, $3, ($4)::vector, $5, $6)",
                        rows,
                    )
                    await conn.execute(
                        """
                        DELETE FROM ego_memory em
                        WHERE em.id IN (
                            SELECT id FROM ego_memory 
                            WHERE user_id = $1 
                            ORDER BY created_at DESC 
                            OFFSET $2
                        )
                        """,
                        user_id, self._max_entries,
                    )
            log.info(f"Added {len(rows)} new entries to memory for user '{user_id}'.")
        except asyncpg.PostgresError as e:
            log.error(f"Database error while adding texts for user '{user_id}': {e}", exc_info=True)

    async def delete_by_log_id(self, log_id: int) -> None:
        """
        Deletes a memory entry by its associated log_id.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot delete by log_id; database pool is not initialized.")
            return
        try:
            async with self._pool.acquire() as conn:
                result = await conn.execute("DELETE FROM ego_memory WHERE log_id = $1", log_id)
                log.info(f"Deletion attempt for log_id: {log_id}. Rows affected: {result.split(' ')[-1]}")
        except asyncpg.PostgresError as e:
            log.error(f"Database error while deleting memory by log_id {log_id}: {e}", exc_info=True)

    async def delete_at_or_after(self, user_id: str, session_id: str, cutoff_log_id: int) -> None:
        """
        Deletes memory rows for a specific user and session that have a log_id
        greater than or equal to a specified cutoff point.

        This is used for response regeneration to clear outdated memories.

        Args:
            user_id: The ID of the user.
            session_id: The ID of the session.
            cutoff_log_id: The log_id from which to start deleting memories.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot delete_at_or_after; database pool is not initialized.")
            return
        try:
            async with self._pool.acquire() as conn:
                await conn.execute(
                    "DELETE FROM ego_memory WHERE user_id = $1 AND session_id = $2 AND log_id IS NOT NULL AND log_id >= $3",
                    str(user_id), str(session_id), int(cutoff_log_id)
                )
            log.info(f"Deleted memory for user '{user_id}', session '{session_id}' at/after log_id {cutoff_log_id}.")
        except (asyncpg.PostgresError, TypeError, ValueError) as e:
            log.error(f"Failed to delete memory for user '{user_id}' at/after log_id {cutoff_log_id}: {e}", exc_info=True)

    async def delete_at_or_after_by_user(self, user_id: str, cutoff_log_id: int) -> None:
        """
        Deletes all memory rows for a user (across all sessions) that have a log_id
        greater than or equal to a specified cutoff point.

        Args:
            user_id: The ID of the user.
            cutoff_log_id: The log_id from which to start deleting memories.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot delete_at_or_after_by_user; database pool is not initialized.")
            return
        try:
            async with self._pool.acquire() as conn:
                await conn.execute(
                    "DELETE FROM ego_memory WHERE user_id = $1 AND log_id IS NOT NULL AND log_id >= $2",
                    str(user_id), int(cutoff_log_id)
                )
            log.info(f"Deleted memory for user '{user_id}' across all sessions at/after log_id {cutoff_log_id}.")
        except (asyncpg.PostgresError, TypeError, ValueError) as e:
            log.error(f"Failed to delete memory for user '{user_id}' at/after log_id {cutoff_log_id}: {e}", exc_info=True)
    
    async def search(
        self,
        user_id: str,
        query: str,
        top_k: int = 5,
        min_score: float = 0.7,
        session_id: Optional[str] = None,
        current_log_id: Optional[int] = None,
    ) -> List[MemoryHit]:
        """
        Searches for relevant memories, reranks them, and applies diversity filtering.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot search; database pool is not initialized.")
            return []

        try:
            qvec = await self.backend.embed(query)
            if len(qvec) != self._dim:
                qvec = (qvec + [0.0] * self._dim)[:self._dim]
            
            async with self._pool.acquire() as conn:
                candidate_k = max(20, top_k * 5)
                records = await conn.fetch(
                    """
                    SELECT session_id, text, (vector <=> ($2)::vector) AS cos_dist,
                           created_at, session_created_at, log_id
                    FROM ego_memory
                    WHERE user_id = $1
                    ORDER BY vector <=> ($2)::vector ASC
                    LIMIT $3
                    """,
                    user_id, self._to_vector_literal(qvec), candidate_k,
                )
        except (Exception, asyncpg.PostgresError) as e:
            log.error(f"Database search failed for user '{user_id}': {e}", exc_info=True)
            return []

        now_utc = datetime.utcnow()
        alpha_age = 0.01
        beta_same = 0.05

        raw_hits: List[MemoryHit] = []
        for r in records or []:
            try:
                score = 1.0 - float(r["cos_dist"])
                log_id = int(r["log_id"]) if r.get("log_id") is not None else None

                if current_log_id is not None and log_id is not None and log_id == current_log_id:
                    continue
                
                created_dt = r["created_at"].replace(tzinfo=None) if r["created_at"] else now_utc
                age_days = (now_utc - created_dt).total_seconds() / 86400.0
                time_penalty = alpha_age * age_days
                
                sess_id = str(r["session_id"]) if r["session_id"] is not None else ""
                session_boost = beta_same if (session_id and sess_id == str(session_id)) else 0.0
                
                adjusted_score = score - time_penalty + session_boost

                log.info(f"Memory candidate: score={adjusted_score:.3f}, min_score={min_score}, text='{str(r['text'])[:100]}...'")
                if adjusted_score >= min_score:
                    raw_hits.append(MemoryHit(
                        user_id=user_id,
                        session_id=sess_id,
                        text=str(r["text"]),
                        score=adjusted_score,
                        created_at=str(r["created_at"]) if r["created_at"] else None,
                        session_created_at=str(r["session_created_at"]) if r["session_created_at"] else None,
                        log_id=log_id,
                    ))
            except (ValueError, TypeError) as e:
                log.warning(f"Could not process a memory record: {dict(r)}, error: {e}")

        def _token_overlap(a: str, b: str) -> float:
            set_a = set(a.lower().split())
            set_b = set(b.lower().split())
            if not set_a or not set_b: return 0.0
            intersection = len(set_a.intersection(set_b))
            union = len(set_a.union(set_b))
            return intersection / union if union > 0 else 0.0

        selected_hits: List[MemoryHit] = []
        raw_hits.sort(key=lambda h: h.score, reverse=True)
        
        log.info(f"Memory search debug: {len(raw_hits)} hits passed min_score filter")
        for i, hit in enumerate(raw_hits):
            log.info(f"Hit {i}: score={hit.score:.3f}, text_preview='{hit.text[:50]}...'")
            is_redundant = any(_token_overlap(hit.text, sel.text) > 0.8 for sel in selected_hits)
            if not is_redundant:
                selected_hits.append(hit)
                log.info(f"Selected hit {i} (total selected: {len(selected_hits)})")
                if len(selected_hits) >= top_k:
                    break
            else:
                log.info(f"Skipped hit {i} due to redundancy")

        log.info(f"Found {len(records)} candidates, selected {len(selected_hits)} diverse memories for user '{user_id}'.")
        return selected_hits

    async def search_for_injection(
        self,
        user_id: str,
        query: str,
        top_k: int = 3,
        session_id: Optional[str] = None,
        current_log_id: Optional[int] = None,
    ) -> List[str]:
        """
        Performs a high-relevance search and returns only the memory texts.
        """
        log.info(f"Performing high-relevance search for context injection for user '{user_id}'.")
        hits = await self.search(
            user_id,
            query,
            top_k=top_k,
            min_score=0.1,
            session_id=session_id,
            current_log_id=current_log_id
        )
        return [hit.text for hit in hits]
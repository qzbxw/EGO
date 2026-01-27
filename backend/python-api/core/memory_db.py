# -----------------------------------------------------------------------------
# --- Library Imports
# -----------------------------------------------------------------------------
import asyncio
import logging
import os
from datetime import datetime
from typing import Any

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
    created_at: str | None = None
    session_created_at: str | None = None
    log_id: int | None = None


# -----------------------------------------------------------------------------
# --- Core Class: VectorMemory
# -----------------------------------------------------------------------------


class VectorMemory:
    """
    Manages the agent's long-term memory using a PostgreSQL vector database.
    """

    def __init__(
        self, backend: LLMProvider, db_url: str | None = None, max_entries_per_user: int = 5000
    ):
        """
        Initializes the VectorMemory instance.
        """
        self.backend = backend
        self.db_url = db_url or os.getenv("DATABASE_URL") or os.getenv("EGO_MEMORY_DB_URL")
        if not self.db_url:
            log.critical(
                "DATABASE_URL or EGO_MEMORY_DB_URL is required for VectorMemory but was not found."
            )
            raise ValueError("A database URL is required for VectorMemory.")

        if "sslmode" not in self.db_url and self.db_url.startswith(
            ("postgres://", "postgresql://")
        ):
            separator = "&" if "?" in self.db_url else "?"
            self.db_url = f"{self.db_url}{separator}sslmode=require"

        self._pool: asyncpg.Pool | None = None
        self._init_lock = asyncio.Lock()
        self._max_entries = max_entries_per_user
        self._dim = 768  # Matches existing pgvector schema (vector(768))
        self._output_dimensionality = 768  # Use 768 dimensions for embeddings to match schema

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
                    log.info("Ensuring 'vector' extension and database schema exist.")
                    await conn.execute("CREATE EXTENSION IF NOT EXISTS vector")

                    # Create embedding cache table for performance optimization
                    await conn.execute(f"""
                        CREATE TABLE IF NOT EXISTS ego_embedding_cache (
                            id BIGSERIAL PRIMARY KEY,
                            text_hash TEXT NOT NULL UNIQUE,
                            text TEXT NOT NULL,
                            embedding VECTOR({self._dim}) NOT NULL,
                            task_type TEXT DEFAULT 'RETRIEVAL_DOCUMENT',
                            created_at TIMESTAMPTZ DEFAULT NOW(),
                            last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
                            access_count INT DEFAULT 1
                        )
                        """)
                    await conn.execute(
                        "CREATE INDEX IF NOT EXISTS ego_embedding_cache_hash_idx ON ego_embedding_cache (text_hash)"
                    )
                    await conn.execute(
                        "CREATE INDEX IF NOT EXISTS ego_embedding_cache_accessed_idx ON ego_embedding_cache (last_accessed_at)"
                    )

                    # Create main memory table with full-text search support
                    await conn.execute(f"""
                        CREATE TABLE IF NOT EXISTS ego_memory (
                            id BIGSERIAL PRIMARY KEY,
                            user_id TEXT NOT NULL,
                            session_id TEXT NOT NULL,
                            text TEXT NOT NULL,
                            vector VECTOR({self._dim}) NOT NULL,
                            created_at TIMESTAMPTZ DEFAULT NOW(),
                            session_created_at TIMESTAMPTZ DEFAULT NOW(),
                            log_id BIGINT NULL UNIQUE,
                            text_tsv tsvector GENERATED ALWAYS AS (to_tsvector('english', text)) STORED
                        )
                        """)
                    await conn.execute(
                        "CREATE INDEX IF NOT EXISTS ego_memory_user_idx ON ego_memory (user_id, created_at DESC)"
                    )
                    await conn.execute(
                        "CREATE INDEX IF NOT EXISTS ego_memory_log_id_idx ON ego_memory (log_id)"
                    )
                    await conn.execute(
                        "CREATE INDEX IF NOT EXISTS ego_memory_text_fts_idx ON ego_memory USING GIN (text_tsv)"
                    )

                    # Drop old IVFFlat index if exists and create HNSW index
                    try:
                        await conn.execute("DROP INDEX IF EXISTS ego_memory_vec_cos_ivfflat")
                        log.info("Dropped old IVFFlat index if it existed.")
                    except asyncpg.PostgresError as e:
                        log.warning(f"Could not drop old IVFFlat index: {e}")

                    try:
                        # HNSW is significantly better for < 1M vectors
                        # m=16 and ef_construction=64 are good defaults for most use cases
                        await conn.execute(
                            "CREATE INDEX IF NOT EXISTS ego_memory_vec_hnsw_idx ON ego_memory USING hnsw (vector vector_cosine_ops) WITH (m = 16, ef_construction = 64)"
                        )
                        log.info("Created HNSW vector index successfully.")
                    except asyncpg.PostgresError as e:
                        log.warning(
                            f"Could not create HNSW vector index (this is non-critical): {e}"
                        )

                log.info("VectorMemory database initialization complete.")
            except (asyncpg.PostgresError, OSError) as e:
                log.critical(
                    f"Failed to initialize VectorMemory database connection: {e}", exc_info=True
                )
                self._pool = None
                raise RuntimeError(f"Database initialization failed: {e}") from e

    def _to_vector_literal(self, vec: list[float]) -> str:
        """Converts a list of floats into a string literal for pgvector."""
        return "[" + ",".join(map(str, vec)) + "]"

    async def _get_cached_embeddings(
        self, texts: list[str], task_type: str = "RETRIEVAL_DOCUMENT"
    ) -> dict[str, list[float] | None]:
        """
        Retrieves cached embeddings for the given texts.

        Returns:
            A dictionary mapping text to its cached embedding (or None if not cached).
        """
        import hashlib

        await self._init()
        if not self._pool:
            return dict.fromkeys(texts)

        # Create hash for each text
        text_to_hash = {text: hashlib.sha256(text.encode("utf-8")).hexdigest() for text in texts}
        hashes = list(text_to_hash.values())

        try:
            async with self._pool.acquire() as conn:
                records = await conn.fetch(
                    """
                    SELECT text_hash, embedding::text
                    FROM ego_embedding_cache
                    WHERE text_hash = ANY($1) AND task_type = $2
                    """,
                    hashes,
                    task_type,
                )

                # Update access stats
                if records:
                    await conn.execute(
                        """
                        UPDATE ego_embedding_cache
                        SET last_accessed_at = NOW(), access_count = access_count + 1
                        WHERE text_hash = ANY($1)
                        """,
                        [r["text_hash"] for r in records],
                    )

                # Build result mapping
                hash_to_embedding = {}
                for r in records:
                    # Parse vector string format: "[1.0,2.0,3.0]"
                    vec_str = r["embedding"].strip("[]")
                    embedding = [float(v) for v in vec_str.split(",")]
                    hash_to_embedding[r["text_hash"]] = embedding

                result = {}
                for text, text_hash in text_to_hash.items():
                    result[text] = hash_to_embedding.get(text_hash)

                return result
        except Exception as e:
            log.warning(f"Failed to retrieve cached embeddings: {e}")
            return dict.fromkeys(texts)

    async def _cache_embeddings(
        self, text_embeddings: dict[str, list[float]], task_type: str = "RETRIEVAL_DOCUMENT"
    ) -> None:
        """
        Caches embeddings for the given texts.

        Args:
            text_embeddings: A dictionary mapping text to its embedding.
            task_type: The task type used for embedding.
        """
        import hashlib

        await self._init()
        if not self._pool or not text_embeddings:
            return

        try:
            rows = []
            for text, embedding in text_embeddings.items():
                text_hash = hashlib.sha256(text.encode("utf-8")).hexdigest()
                rows.append([text_hash, text, self._to_vector_literal(embedding), task_type])

            async with self._pool.acquire() as conn:
                await conn.executemany(
                    """
                    INSERT INTO ego_embedding_cache (text_hash, text, embedding, task_type)
                    VALUES ($1, $2, ($3)::vector, $4)
                    ON CONFLICT (text_hash) DO UPDATE
                    SET last_accessed_at = NOW(), access_count = ego_embedding_cache.access_count + 1
                    """,
                    rows,
                )
            log.info(f"Cached {len(rows)} embeddings.")
        except Exception as e:
            log.warning(f"Failed to cache embeddings: {e}")

    async def add_texts(
        self,
        user_id: str,
        session_id: str,
        texts: list[str],
        log_id: int | None = None,
        session_created_at: str | None = None,
    ) -> None:
        """
        Embeds and adds a list of texts to the user's memory using batch embeddings and cache, then prunes old entries.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot add texts; database pool is not initialized.")
            return

        # Filter out empty texts
        valid_texts = [text.strip() for text in texts if text and text.strip()]
        if not valid_texts:
            log.info(f"No valid texts to add to memory for user '{user_id}'.")
            return

        session_dt = None
        if session_created_at:
            try:
                session_dt = datetime.fromisoformat(session_created_at.replace("Z", "+00:00"))
            except (ValueError, TypeError) as e:
                log.warning(f"Failed to parse session_created_at '{session_created_at}': {e}")

        try:
            # Check cache first
            task_type = "RETRIEVAL_DOCUMENT"
            cached = await self._get_cached_embeddings(valid_texts, task_type)

            # Identify texts that need embedding
            texts_to_embed = [text for text in valid_texts if cached.get(text) is None]

            # Batch embed uncached texts
            new_embeddings = {}
            if texts_to_embed:
                if hasattr(self.backend, "batch_embed"):
                    embeddings = await self.backend.batch_embed(
                        texts_to_embed, task_type, self._dim
                    )
                    new_embeddings = dict(zip(texts_to_embed, embeddings))
                else:
                    # Fallback to sequential embedding
                    for text in texts_to_embed:
                        vec = await self.backend.embed(text, task_type, self._dim)
                        new_embeddings[text] = vec

                # Cache the new embeddings
                if new_embeddings:
                    await self._cache_embeddings(new_embeddings, task_type)

            # Build rows for insertion
            rows: list[list[Any]] = []
            for text in valid_texts:
                res = cached.get(text) or new_embeddings.get(text)
                if res is not None:
                    v_list: list[float] = res
                    # Ensure correct dimensionality
                    if len(v_list) != self._dim:
                        v_list = (v_list + [0.0] * self._dim)[: self._dim]

                    rows.append(
                        [
                            user_id,
                            str(session_id),
                            text,
                            self._to_vector_literal(v_list),
                            log_id,
                            session_dt,
                        ]
                    )

            if not rows:
                log.warning(f"No valid embeddings to add to memory for user '{user_id}'.")
                return

            # Insert into database and prune old entries
            async with self._pool.acquire() as conn, conn.transaction():
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
                    user_id,
                    self._max_entries,
                )
            log.info(
                f"Added {len(rows)} new entries to memory for user '{user_id}' (cache hits: {len([t for t in valid_texts if cached.get(t)])})."
            )
        except Exception as e:
            log.error(f"Error while adding texts for user '{user_id}': {e}", exc_info=True)

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
                log.info(
                    f"Deletion attempt for log_id: {log_id}. Rows affected: {result.split(' ')[-1]}"
                )
        except asyncpg.PostgresError as e:
            log.error(
                f"Database error while deleting memory by log_id {log_id}: {e}", exc_info=True
            )

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
                    str(user_id),
                    str(session_id),
                    int(cutoff_log_id),
                )
            log.info(
                f"Deleted memory for user '{user_id}', session '{session_id}' at/after log_id {cutoff_log_id}."
            )
        except (asyncpg.PostgresError, TypeError, ValueError) as e:
            log.error(
                f"Failed to delete memory for user '{user_id}' at/after log_id {cutoff_log_id}: {e}",
                exc_info=True,
            )

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
                    str(user_id),
                    int(cutoff_log_id),
                )
            log.info(
                f"Deleted memory for user '{user_id}' across all sessions at/after log_id {cutoff_log_id}."
            )
        except (asyncpg.PostgresError, TypeError, ValueError) as e:
            log.error(
                f"Failed to delete memory for user '{user_id}' at/after log_id {cutoff_log_id}: {e}",
                exc_info=True,
            )

    async def search(
        self,
        user_id: str,
        query: str,
        top_k: int = 5,
        min_score: float = 0.7,
        session_id: str | None = None,
        current_log_id: int | None = None,
        semantic_weight: float = 0.7,
        fts_weight: float = 0.3,
    ) -> list[MemoryHit]:
        """
        Hybrid search combining semantic (vector) and full-text (FTS) search.

        Args:
            user_id: User ID to search memories for.
            query: Search query text.
            top_k: Number of results to return.
            min_score: Minimum score threshold for filtering results.
            session_id: Optional session ID for boosting same-session results.
            current_log_id: Optional log ID to exclude from results.
            semantic_weight: Weight for semantic search (default 0.7).
            fts_weight: Weight for full-text search (default 0.3).

        Returns:
            List of MemoryHit objects, ranked and deduplicated.
        """
        await self._init()
        if not self._pool:
            log.error("Cannot search; database pool is not initialized.")
            return []

        try:
            # Embed query using RETRIEVAL_QUERY task type
            qvec = await self.backend.embed(
                query, task_type="RETRIEVAL_QUERY", output_dimensionality=self._dim
            )
            if len(qvec) != self._dim:
                qvec = (qvec + [0.0] * self._dim)[: self._dim]

            async with self._pool.acquire() as conn:
                candidate_k = max(20, top_k * 5)

                # Hybrid search: combine semantic and FTS results
                records = await conn.fetch(
                    """
                    WITH semantic_results AS (
                        SELECT
                            session_id, text,
                            (1.0 - (vector <=> ($2)::vector)) AS semantic_score,
                            created_at, session_created_at, log_id
                        FROM ego_memory
                        WHERE user_id = $1
                        ORDER BY vector <=> ($2)::vector ASC
                        LIMIT $3
                    ),
                    fts_results AS (
                        SELECT
                            session_id, text,
                            ts_rank(text_tsv, websearch_to_tsquery('english', $4)) AS fts_score,
                            created_at, session_created_at, log_id
                        FROM ego_memory
                        WHERE user_id = $1
                          AND text_tsv @@ websearch_to_tsquery('english', $4)
                        ORDER BY fts_score DESC
                        LIMIT $3
                    )
                    SELECT
                        COALESCE(s.session_id, f.session_id) AS session_id,
                        COALESCE(s.text, f.text) AS text,
                        COALESCE(s.created_at, f.created_at) AS created_at,
                        COALESCE(s.session_created_at, f.session_created_at) AS session_created_at,
                        COALESCE(s.log_id, f.log_id) AS log_id,
                        (COALESCE(s.semantic_score, 0.0) * $5 + COALESCE(f.fts_score, 0.0) * $6) AS combined_score
                    FROM semantic_results s
                    FULL OUTER JOIN fts_results f
                      ON s.text = f.text AND s.session_id = f.session_id
                    ORDER BY combined_score DESC
                    LIMIT $3
                    """,
                    user_id,
                    self._to_vector_literal(qvec),
                    candidate_k,
                    query,  # For FTS
                    semantic_weight,
                    fts_weight,
                )
        except (Exception, asyncpg.PostgresError) as e:
            log.error(
                f"Hybrid search failed for user '{user_id}': {e}. Falling back to semantic-only.",
                exc_info=True,
            )
            # Fallback to semantic-only search
            try:
                async with self._pool.acquire() as conn:
                    candidate_k = max(20, top_k * 5)
                    records = await conn.fetch(
                        """
                        SELECT session_id, text, (1.0 - (vector <=> ($2)::vector)) AS combined_score,
                               created_at, session_created_at, log_id
                        FROM ego_memory
                        WHERE user_id = $1
                        ORDER BY vector <=> ($2)::vector ASC
                        LIMIT $3
                        """,
                        user_id,
                        self._to_vector_literal(qvec),
                        candidate_k,
                    )
            except (Exception, asyncpg.PostgresError) as e2:
                log.error(
                    f"Fallback semantic search also failed for user '{user_id}': {e2}",
                    exc_info=True,
                )
                return []

        now_utc = datetime.utcnow()
        alpha_age = 0.01
        beta_same = 0.05

        raw_hits: list[MemoryHit] = []
        for r in records or []:
            try:
                # Use combined_score from hybrid search
                score = float(r.get("combined_score", 0.0))
                log_id = int(r["log_id"]) if r.get("log_id") is not None else None

                if current_log_id is not None and log_id is not None and log_id == current_log_id:
                    continue

                created_dt = r["created_at"].replace(tzinfo=None) if r["created_at"] else now_utc
                age_days = (now_utc - created_dt).total_seconds() / 86400.0
                time_penalty = alpha_age * age_days

                sess_id = str(r["session_id"]) if r["session_id"] is not None else ""
                session_boost = beta_same if (session_id and sess_id == str(session_id)) else 0.0

                adjusted_score = score - time_penalty + session_boost

                log.info(
                    f"Memory candidate: score={adjusted_score:.3f}, min_score={min_score}, text='{str(r['text'])[:100]}...'"
                )
                if adjusted_score >= min_score:
                    raw_hits.append(
                        MemoryHit(
                            user_id=user_id,
                            session_id=sess_id,
                            text=str(r["text"]),
                            score=adjusted_score,
                            created_at=str(r["created_at"]) if r["created_at"] else None,
                            session_created_at=(
                                str(r["session_created_at"]) if r["session_created_at"] else None
                            ),
                            log_id=log_id,
                        )
                    )
            except (ValueError, TypeError) as e:
                log.warning(f"Could not process a memory record: {dict(r)}, error: {e}")

        def _token_overlap(a: str, b: str) -> float:
            set_a = set(a.lower().split())
            set_b = set(b.lower().split())
            if not set_a or not set_b:
                return 0.0
            intersection = len(set_a.intersection(set_b))
            union = len(set_a.union(set_b))
            return intersection / union if union > 0 else 0.0

        selected_hits: list[MemoryHit] = []
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

        log.info(
            f"Found {len(records)} candidates, selected {len(selected_hits)} diverse memories for user '{user_id}'."
        )
        return selected_hits

    async def search_for_injection(
        self,
        user_id: str,
        query: str,
        top_k: int = 3,
        session_id: str | None = None,
        current_log_id: int | None = None,
    ) -> list[str]:
        """
        Performs a high-relevance search and returns only the memory texts.
        Uses higher min_score threshold (0.7) to ensure quality results.
        """
        log.info(f"Performing high-relevance search for context injection for user '{user_id}'.")
        hits = await self.search(
            user_id,
            query,
            top_k=top_k,
            min_score=0.7,
            session_id=session_id,
            current_log_id=current_log_id,
        )
        return [hit.text for hit in hits]

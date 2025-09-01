export interface User {
	id: number;
	username: string;
	role: 'user';
	created_at: string;
}
export interface AuthResponse {
	access_token: string;
	refresh_token: string;
	user: User;
}
export interface FileAttachment {
	file_name: string;
	mime_type: string;
	preview_url?: string;
}
export interface ChatSession {
	uuid: string;
	title: string;
	mode: string;
	custom_instructions: string | null;
	created_at: string;
}
export interface HistoryLog {
	id: number;
	user_query: string;
	final_response: string | null;
	timestamp: string;
	attachments: FileAttachment[];
}
export interface ChatMessage {
	id: number;
	author: 'user' | 'ego';
	text: string;
	isThinking?: boolean;
	isStreaming?: boolean;
	isCancelled?: boolean;
	attachments?: FileAttachment[];
	logId?: number;
}
export interface FilePayload {
	base64_data: string;
	mime_type: string;
	file_name: string;
}
export interface CachedFile {
	file_name: string;
	mime_type: string;
	uri: string;
}
export interface WebSocketSendPayload {
	query: string;
	mode: string;
	session_uuid: string | null;
	files?: (FilePayload | CachedFile)[];
	is_regeneration: boolean;
	custom_instructions?: string;
	request_log_id_to_regen?: number;
	temp_id?: number;
	memory_enabled?: boolean;
	upload_id?: string;
}
interface WsEventBase {
	type: string;
}
export interface WsErrorEvent extends WsEventBase {
	type: 'error';
	data: {
		message: string;
	};
}
export interface WsPongEvent extends WsEventBase {
	type: 'pong';
}
export interface WsSessionCreatedEvent extends WsEventBase {
	type: 'session_created';
	data: ChatSession;
}
export interface WsLogCreatedEvent extends WsEventBase {
	type: 'log_created';
	data: {
		log_id: number;
		temp_id: number;
	};
}
export interface WsThoughtHeaderEvent extends WsEventBase {
	type: 'thought_header';
	data: {
		header: string;
	};
}

export interface WsToolProgressEvent extends WsEventBase {
	type: 'tool_progress';
	data: {
		tool_name: string;
		progress: string;
		header: string;
	};
}
export interface WsChunkEvent extends WsEventBase {
	type: 'chunk';
	data: {
		text: string;
	};
}
export interface WsSessionTitleUpdatedEvent extends WsEventBase {
	type: 'session_title_updated';
	data: {
		uuid: string;
		title: string;
	};
}
export interface WsUsageUpdateEvent extends WsEventBase {
	type: 'usage_update';
	data: unknown;
}
export interface WsDoneEvent extends WsEventBase {
	type: 'done';
}
export type WsEvent =
	| WsErrorEvent
	| WsPongEvent
	| WsSessionCreatedEvent
	| WsLogCreatedEvent
	| WsThoughtHeaderEvent
	| WsToolProgressEvent
	| WsChunkEvent
	| WsSessionTitleUpdatedEvent
	| WsUsageUpdateEvent
	| WsDoneEvent;
export interface PublicStats {
    total_tokens: number;
    total_sessions: number;
    total_requests: number;
    avg_response_time_ms: number;
    days_active: number;
    total_files_uploaded: number;
    total_thinking_iterations: number;
}
export interface ValidateApiKeyResponse {
    is_valid: boolean;
}
export interface ListModelsResponse {
    models: string[];
}
export interface CredentialResponse {
    credential?: string;
    select_by?: 'auto' | 'user' | 'user_1tap' | 'user_2tap' | 'btn' | 'btn_confirm' | 'btn_add_session' | 'btn_confirm_add_session';
    clientId?: string;
}
export interface UserStats {
    user_id: number;
    username: string;
    total_requests: number;
    total_sessions: number;
    total_files_uploaded: number;
    total_files_size_mb: number;
    total_prompt_tokens: number;
    total_completion_tokens: number;
    total_tokens: number;
    avg_response_time_ms: number;
    min_response_time_ms: number;
    max_response_time_ms: number;
    total_thinking_iterations: number;
    avg_thinking_iterations: number;
    max_thinking_iterations: number;
    total_regenerations: number;
    memory_enabled_requests: number;
    memory_usage_percent: number;
    provider_usage: Record<string, number>;
    model_usage: Record<string, number>;
    first_request_at: string | null;
    last_request_at: string | null;
    days_active: number;
    requests_per_day: number;
    recent_activity: Array<{
        date: string;
        requests: number;
        tokens: number;
    }>;
}
export interface ProviderTokens {
    llm_provider: string;
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
}
export type ThemeName = 'dark' | 'light';
export type ChatMode = 'default' | 'deeper' | 'research' | 'agent';
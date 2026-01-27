export interface ToolAction {
	callId: string;
	toolName: string;
	status: 'running' | 'completed' | 'failed';
	progress?: string;
	header?: string;
}

export interface StreamStoreState {
	thoughtHeader: string;
	thoughts: ThoughtStep[];
	activeTools: ToolAction[];
	textStream: string;
	isDone: boolean;
	error: string;
	sessionUUID: string | null;
	newlyCreatedSessionUUID: string | null;
	lastUserMessage: { temp_id: number; log_id: number } | null;
	wasCancelled: boolean;
	currentLogId: number | null;
	isRecovering: boolean;
	recoveredFromRest: boolean;
	lastSeqNumber: number;
	activePlan: SessionPlan | null;
}

export interface User {
	id: number;
	username: string;
	role: 'user';
	created_at: string;
	llm_provider?: string;
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
	thoughts?: BackendThoughtStep[];
	timestamp: string;
	attachments: FileAttachment[];
}

export interface BackendThoughtStep extends ThoughtStep {
	thoughts_header?: string;
	tool_reasoning?: string;
	thoughts?: string;
	tool_name?: string;
}

export interface ThoughtStep {
	id: string;
	type: 'thought' | 'tool';
	header: string;
	content?: string;
	reasoning?: string;
	status?: 'running' | 'completed' | 'failed';
	toolName?: string;
	progress?: string;
	confidence_score?: number;
	self_critique?: string;
	plan_status?: string;
}

export interface ChatMessage {
	id: number;
	author: 'user' | 'ego';
	text: string;
	thoughts?: ThoughtStep[]; // Chain of thought steps
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
	cached_files?: CachedFile[];
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
		call_id?: string;
	};
}
export interface WsChunkEvent extends WsEventBase {
	type: 'chunk';
	data: {
		text: string;
		/** Sequence number to prevent duplicates and ensure ordering */
		seq?: number;
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

export interface WsProcessResumedEvent extends WsEventBase {
	type: 'process_resumed';
	data: {
		session_uuid: string;
		log_id: number;
		started_at: string;
	};
}

export interface WsTextStreamResetEvent extends WsEventBase {
	type: 'text_stream_reset';
	data: Record<string, never>;
}

export interface PlanStep {
	id: number;
	plan_id: number;
	description: string;
	status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'skipped';
	step_order: number;
}

export interface SessionPlan {
	id: number;
	session_uuid: string;
	title: string;
	is_active: boolean;
	steps: PlanStep[];
}

export interface WsPlanUpdatedEvent extends WsEventBase {
	type: 'plan_updated';
	data: SessionPlan;
}

export interface WsThoughtEvent extends WsEventBase {
	type: 'thought';
	data: {
		thoughts?: string;
		content?: string;
		tool_reasoning?: string;
		thoughts_header?: string;
		confidence_score?: number;
		self_critique?: string;
		plan_status?: string;
	};
}

export interface WsToolCallEvent extends WsEventBase {
	type: 'tool_call';
	data: {
		tool_name: string;
		call_id?: string;
	};
}

export interface WsToolOutputEvent extends WsEventBase {
	type: 'tool_output';
	data: {
		tool_name: string;
		call_id?: string;
		output: string;
	};
}

export interface WsToolErrorEvent extends WsEventBase {
	type: 'tool_error';
	data: {
		tool_name: string;
		call_id?: string;
		error: string;
	};
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
	| WsDoneEvent
	| WsProcessResumedEvent
	| WsTextStreamResetEvent
	| WsPlanUpdatedEvent
	| WsThoughtEvent
	| WsToolCallEvent
	| WsToolOutputEvent
	| WsToolErrorEvent;

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
	select_by?:
		| 'auto'
		| 'user'
		| 'user_1tap'
		| 'user_2tap'
		| 'btn'
		| 'btn_confirm'
		| 'btn_add_session'
		| 'btn_confirm_add_session';
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

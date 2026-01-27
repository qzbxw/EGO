<script module lang="ts">
	export interface ExtendedFile extends File {
		uploadStatus?: 'pending' | 'uploading' | 'done' | 'error';
		uploadId?: string;
		previewUrl?: string;
	}
</script>

<script lang="ts">
	import { Paperclip } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { toast } from 'svelte-sonner';
	import { api } from '$lib/api';
	import { chatStore } from '$lib/stores/chat.svelte';

	let { attachedFiles = $bindable([]) }: { attachedFiles: ExtendedFile[] } = $props();

	function addFiles(fileList: FileList | File[]) {
		const incoming = Array.from(fileList);

		// Removed strict extension filtering. Now filtering logic is effectively "allow all",
		// but we still want to give some feedback or just process everything.
		// The prompt requested: "Implement loading of all types of files".
		// We will allow all files to pass through.
		// The validation for Video/Audio for specific providers will happen in ChatContainer/ChatInput
		// or we can add it here if we know the provider. But ChatContainer is better suited for that
		// as it holds the state.

		const validFiles: ExtendedFile[] = [];
		const MAX_FILE_SIZE = 500 * 1024 * 1024; // 500 MB (matching backend)
		const MAX_TOTAL_SIZE = 1000 * 1024 * 1024; // 1 GB (matching backend)

		let currentTotalSize = attachedFiles.reduce((sum, file) => sum + file.size, 0);

		for (const file of incoming) {
			if (file.size > MAX_FILE_SIZE) {
				toast.error(
					$_('chat.file_too_large') || `Файл ${file.name} слишком большой (максимум 500 МБ)`
				);
				continue;
			}
			if (currentTotalSize + file.size > MAX_TOTAL_SIZE) {
				toast.error($_('chat.total_size_exceeded') || 'Общий размер файлов превышает 1 ГБ');
				break;
			}

			// Augment with status
			const extFile = file as ExtendedFile;
			extFile.uploadStatus = 'pending';

			// Create preview URL only for images/videos that browsers support?
			// URL.createObjectURL works for any blob, but displaying it depends on the tag.
			extFile.previewUrl = URL.createObjectURL(file);

			validFiles.push(extFile);
			currentTotalSize += file.size;
		}

		if (attachedFiles.length + validFiles.length > 10) {
			toast.warning($_('chat.max_files_exceeded') || 'Максимум 10 файлов');
		}

		const newFiles = validFiles.slice(0, Math.max(0, 10 - attachedFiles.length));
		attachedFiles = [...attachedFiles, ...newFiles];

		// Trigger upload for new files
		newFiles.forEach(uploadFile);
	}

	async function uploadFile(file: ExtendedFile) {
		if (file.uploadStatus === 'uploading' || file.uploadStatus === 'done') return;

		file.uploadStatus = 'uploading';
		// Trigger reactivity
		attachedFiles = [...attachedFiles];

		try {
			const formData = new FormData();
			formData.append('files', file);
			// Pass current session UUID if available to associate immediately
			const sessionId = chatStore.stream.sessionUUID || chatStore.messagesSessionUUID;
			if (sessionId) {
				formData.append('session_uuid', sessionId);
			}

			const res = await api.postMultipart<{ files: { upload_id: string; file_name: string }[] }>(
				'/upload',
				formData
			);

			if (res && res.files && res.files.length > 0) {
				file.uploadId = res.files[0].upload_id;
				file.uploadStatus = 'done';
			} else {
				throw new Error('No upload ID returned');
			}
		} catch (e) {
			console.error('Upload failed:', e);
			file.uploadStatus = 'error';
			toast.error(`Upload failed for ${file.name}`);
		} finally {
			attachedFiles = [...attachedFiles];
		}
	}

	let handleFileSelect = $state((event: Event) => {
		const target = event.target as HTMLInputElement;
		if (target.files) {
			addFiles(target.files);
		}
		// Reset input value to allow selecting the same file again if needed
		target.value = '';
	});

	function removeFile(index: number) {
		const file = attachedFiles[index];
		if (file.previewUrl) URL.revokeObjectURL(file.previewUrl);
		attachedFiles.splice(index, 1);
		attachedFiles = [...attachedFiles];
	}
	export { addFiles, handleFileSelect, removeFile };
</script>

<label
	class="group flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl transition-all duration-200 hover:bg-white/10 active:scale-95"
	title={$_('chat.attach_files')}
>
	<Paperclip class="h-5 w-5 text-text-secondary transition-colors group-hover:text-text-primary" />
	<!-- Removed accept attribute to allow all files -->
	<input type="file" multiple onchange={handleFileSelect} class="hidden" />
</label>

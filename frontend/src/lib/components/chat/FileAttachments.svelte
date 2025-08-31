<script lang="ts">
	import { Paperclip, X } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { toast } from 'svelte-sonner';
	let { attachedFiles = $bindable([]) }: { attachedFiles: File[] } = $props();
	function addFiles(fileList: FileList | File[]) {
		const incoming = Array.from(fileList);
		const allowedExt = ['png','jpg','jpeg','gif','webp','bmp','heic','heif'];
		const images = incoming.filter(f => {
			if (!f) return false;
			const t = typeof f.type === 'string' ? f.type : '';
			if (t && t.startsWith('image/')) return true;
			const name = (f as File).name || '';
			const ext = name.split('.').pop()?.toLowerCase() || '';
			return allowedExt.includes(ext);
		});
		const rejected = incoming.length - images.length;
		if (rejected > 0) {
			toast.warning($_('chat.only_images_allowed') || 'Допускаются только изображения');
		}
		const MAX_FILE_SIZE = 50 * 1024 * 1024;
		const MAX_TOTAL_SIZE = 100 * 1024 * 1024;
		const validImages = [];
		let currentTotalSize = attachedFiles.reduce((sum, file) => sum + file.size, 0);
		for (const image of images) {
			if (image.size > MAX_FILE_SIZE) {
				toast.error($_('chat.file_too_large') || `Файл ${image.name} слишком большой (максимум 50 МБ)`);
				continue;
			}
			if (currentTotalSize + image.size > MAX_TOTAL_SIZE) {
				toast.error($_('chat.total_size_exceeded') || 'Общий размер файлов превышает 100 МБ');
				break;
			}
			validImages.push(image);
			currentTotalSize += image.size;
		}
		if (attachedFiles.length + validImages.length > 10) {
			toast.warning($_('chat.max_files_exceeded'));
		}
		attachedFiles = [
			...attachedFiles,
			...validImages.slice(0, Math.max(0, 10 - attachedFiles.length))
		];
	}
	function handleFileSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files) {
			addFiles(target.files);
		}
	}
	function removeFile(index: number) {
		attachedFiles.splice(index, 1);
		attachedFiles = [...attachedFiles];
	}
	export { addFiles, handleFileSelect, removeFile };
</script>
<label class="p-2 rounded-md hover:bg-tertiary cursor-pointer transition-colors" title={$_('chat.attach_files')}>
	<Paperclip class="w-5 h-5 text-text-secondary" />
	<input
		type="file"
		multiple
		onchange={handleFileSelect}
		class="hidden"
		accept="image/*"
	/>
</label>
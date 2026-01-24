export interface CompressionOptions {
	maxWidth?: number;
	maxHeight?: number;
	quality?: number;
	outputFormat?: 'image/jpeg' | 'image/webp' | 'image/png';
}

const DEFAULT_OPTIONS: CompressionOptions = {
	maxWidth: 1920,
	maxHeight: 1080,
	quality: 0.85,
	outputFormat: 'image/jpeg'
};

export async function compressImage(file: File, options: CompressionOptions = {}): Promise<File> {
	if (!file.type.startsWith('image/')) {
		return file;
	}

	// Skip compression for HEIC/HEVC files as they are often not supported by canvas
	// or are already highly compressed. Let backend handle or reject if unsupported.
	if (
		file.type === 'image/gif' ||
		file.type === 'image/svg+xml' ||
		file.type === 'image/heic' ||
		file.type === 'image/heif' ||
		file.name.toLowerCase().endsWith('.hevc')
	) {
		return file;
	}

	const opts = { ...DEFAULT_OPTIONS, ...options };

	return new Promise((resolve, reject) => {
		const img = new Image();
		const url = URL.createObjectURL(file);

		img.onload = () => {
			URL.revokeObjectURL(url);

			try {
				let { width, height } = img;
				const maxWidth = opts.maxWidth!;
				const maxHeight = opts.maxHeight!;

				if (width > maxWidth || height > maxHeight) {
					const ratio = Math.min(maxWidth / width, maxHeight / height);
					width = Math.round(width * ratio);
					height = Math.round(height * ratio);
				}

				const canvas = document.createElement('canvas');
				canvas.width = width;
				canvas.height = height;

				const ctx = canvas.getContext('2d');
				if (!ctx) {
					reject(new Error('Failed to get canvas context'));
					return;
				}

				ctx.imageSmoothingEnabled = true;
				ctx.imageSmoothingQuality = 'high';
				ctx.drawImage(img, 0, 0, width, height);

				canvas.toBlob(
					(blob) => {
						if (!blob) {
							reject(new Error('Compression failed: canvas.toBlob returned null'));
							return;
						}
						if (blob.size >= file.size) {
							resolve(file);
							return;
						}

						const originalName = file.name;
						const extension =
							opts.outputFormat === 'image/jpeg'
								? 'jpg'
								: opts.outputFormat === 'image/webp'
									? 'webp'
									: 'png';
						const nameWithoutExt = originalName.replace(/\.[^/.]+$/, '');
						const newName = `${nameWithoutExt}_compressed.${extension}`;

						const compressedFile = new File([blob], newName, {
							type: opts.outputFormat!,
							lastModified: Date.now()
						});

						console.log(
							`[ImageCompression] ${originalName}: ${(file.size / 1024).toFixed(1)}KB → ${(compressedFile.size / 1024).toFixed(1)}KB (${Math.round((1 - compressedFile.size / file.size) * 100)}% reduction)`
						);

						resolve(compressedFile);
					},
					opts.outputFormat!,
					opts.quality!
				);
			} catch (error) {
				reject(error);
			}
		};

		img.onerror = () => {
			URL.revokeObjectURL(url);
			reject(new Error('Failed to load image'));
		};

		img.src = url;
	});
}

export async function compressImages(
	files: File[],
	options: CompressionOptions = {}
): Promise<File[]> {
	return Promise.all(files.map((file) => compressImage(file, options)));
}

export function shouldCompress(file: File): boolean {
	if (!file.type.startsWith('image/')) {
		return false;
	}

	if (
		file.type === 'image/gif' ||
		file.type === 'image/svg+xml' ||
		file.type === 'image/heic' ||
		file.type === 'image/heif' ||
		file.name.toLowerCase().endsWith('.hevc')
	) {
		return false;
	}

	const MIN_SIZE_TO_COMPRESS = 200 * 1024;
	return file.size > MIN_SIZE_TO_COMPRESS;
}

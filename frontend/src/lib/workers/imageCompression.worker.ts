/**
 * Web Worker для компрессии изображений
 * Перемещает тяжелые операции по обработке изображений в отдельный поток
 */

interface CompressionOptions {
	maxWidth?: number;
	maxHeight?: number;
	quality?: number;
	outputFormat?: 'image/jpeg' | 'image/webp' | 'image/png';
}

interface CompressionMessage {
	id: string;
	file: {
		data: ArrayBuffer;
		name: string;
		type: string;
		size: number;
	};
	options?: CompressionOptions;
}

interface CompressionResponse {
	id: string;
	success: boolean;
	file?: {
		data: ArrayBuffer;
		name: string;
		type: string;
		size: number;
	};
	error?: string;
	compressionRatio?: number;
}

const DEFAULT_OPTIONS: Required<CompressionOptions> = {
	maxWidth: 1920,
	maxHeight: 1080,
	quality: 0.85,
	outputFormat: 'image/jpeg'
};

async function compressImageInWorker(
	fileData: ArrayBuffer,
	fileName: string,
	fileType: string,
	options: CompressionOptions = {}
): Promise<{
	data: ArrayBuffer;
	name: string;
	type: string;
	size: number;
	compressionRatio: number;
}> {
	if (!fileType.startsWith('image/')) {
		throw new Error('Not an image file');
	}

	if (fileType === 'image/gif' || fileType === 'image/svg+xml') {
		throw new Error('GIF and SVG compression not supported');
	}

	const opts = { ...DEFAULT_OPTIONS, ...options };
	const blob = new Blob([fileData], { type: fileType });
	const imageBitmap = await createImageBitmap(blob);

	let { width, height } = imageBitmap;
	const maxWidth = opts.maxWidth;
	const maxHeight = opts.maxHeight;

	if (width > maxWidth || height > maxHeight) {
		const ratio = Math.min(maxWidth / width, maxHeight / height);
		width = Math.round(width * ratio);
		height = Math.round(height * ratio);
	}

	const canvas = new OffscreenCanvas(width, height);
	const ctx = canvas.getContext('2d');

	if (!ctx) {
		throw new Error('Failed to get canvas context');
	}

	ctx.imageSmoothingEnabled = true;
	ctx.imageSmoothingQuality = 'high';
	ctx.drawImage(imageBitmap, 0, 0, width, height);

	const compressedBlob = await canvas.convertToBlob({
		type: opts.outputFormat,
		quality: opts.quality
	});

	imageBitmap.close();

	const originalSize = fileData.byteLength;
	const compressedSize = compressedBlob.size;

	// Если сжатое изображение больше оригинала, возвращаем оригинал
	if (compressedSize >= originalSize) {
		return {
			data: fileData,
			name: fileName,
			type: fileType,
			size: originalSize,
			compressionRatio: 0
		};
	}

	const extension =
		opts.outputFormat === 'image/jpeg'
			? 'jpg'
			: opts.outputFormat === 'image/webp'
				? 'webp'
				: 'png';
	const nameWithoutExt = fileName.replace(/\.[^/.]+$/, '');
	const newName = `${nameWithoutExt}_compressed.${extension}`;

	const compressedData = await compressedBlob.arrayBuffer();
	const compressionRatio = Math.round((1 - compressedSize / originalSize) * 100);

	return {
		data: compressedData,
		name: newName,
		type: opts.outputFormat,
		size: compressedSize,
		compressionRatio
	};
}

self.onmessage = async (e: MessageEvent<CompressionMessage>) => {
	const { id, file, options } = e.data;

	try {
		const result = await compressImageInWorker(file.data, file.name, file.type, options);

		const response: CompressionResponse = {
			id,
			success: true,
			file: {
				data: result.data,
				name: result.name,
				type: result.type,
				size: result.size
			},
			compressionRatio: result.compressionRatio
		};

		self.postMessage(response, [result.data]);
	} catch (error) {
		const response: CompressionResponse = {
			id,
			success: false,
			error: error instanceof Error ? error.message : 'Unknown error'
		};

		self.postMessage(response);
	}
};

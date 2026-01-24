/**
 * Обертка для использования Web Worker компрессии изображений
 */

import type { CompressionOptions } from './imageCompression';

interface PendingCompression {
	resolve: (file: File) => void;
	reject: (error: Error) => void;
}

class ImageCompressionWorkerManager {
	private worker: Worker | null = null;
	private pendingTasks = new Map<string, PendingCompression>();
	private taskIdCounter = 0;

	private ensureWorker(): Worker {
		if (!this.worker) {
			this.worker = new Worker(new URL('../workers/imageCompression.worker.ts', import.meta.url), {
				type: 'module'
			});

			this.worker.onmessage = (e) => {
				const { id, success, file, error, compressionRatio } = e.data;
				const pending = this.pendingTasks.get(id);

				if (!pending) {
					console.warn('[ImageCompressionWorker] Received response for unknown task:', id);
					return;
				}

				this.pendingTasks.delete(id);

				if (success && file) {
					const resultFile = new File([file.data], file.name, {
						type: file.type,
						lastModified: Date.now()
					});

					if (compressionRatio !== undefined && compressionRatio > 0) {
						console.log(`[ImageCompressionWorker] ${file.name}: ${compressionRatio}% reduction`);
					}

					pending.resolve(resultFile);
				} else {
					pending.reject(new Error(error || 'Compression failed'));
				}
			};

			this.worker.onerror = (error) => {
				console.error('[ImageCompressionWorker] Worker error:', error);
				// Отклоняем все ожидающие задачи
				for (const [id, pending] of this.pendingTasks.entries()) {
					pending.reject(new Error('Worker error'));
					this.pendingTasks.delete(id);
				}
			};
		}

		return this.worker;
	}

	async compressImage(file: File, options?: CompressionOptions): Promise<File> {
		// Проверяем, нужно ли сжимать файл
		if (!file.type.startsWith('image/')) {
			return file;
		}

		if (file.type === 'image/gif' || file.type === 'image/svg+xml') {
			return file;
		}

		const worker = this.ensureWorker();
		const id = `task-${++this.taskIdCounter}`;

		return new Promise<File>((resolve, reject) => {
			this.pendingTasks.set(id, { resolve, reject });

			file
				.arrayBuffer()
				.then((data) => {
					worker.postMessage(
						{
							id,
							file: {
								data,
								name: file.name,
								type: file.type,
								size: file.size
							},
							options
						},
						[data]
					);
				})
				.catch(reject);
		});
	}

	async compressImages(files: File[], options?: CompressionOptions): Promise<File[]> {
		return Promise.all(files.map((file) => this.compressImage(file, options)));
	}

	terminate(): void {
		if (this.worker) {
			this.worker.terminate();
			this.worker = null;
		}
		this.pendingTasks.clear();
	}
}

export const imageCompressionWorker = new ImageCompressionWorkerManager();

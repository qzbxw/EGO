export interface CanvasState {
	isOpen: boolean;
	content: string;
	language: string;
	title: string;
}

class CanvasStore {
	isOpen = $state(false);
	content = $state('');
	language = $state('');
	title = $state('');

	open(content: string, language: string, title: string = 'Canvas') {
		this.content = content;
		this.language = language;
		this.title = title;
		this.isOpen = true;
	}

	close() {
		this.isOpen = false;
	}

	toggle() {
		this.isOpen = !this.isOpen;
	}
}

export const canvasStore = new CanvasStore();

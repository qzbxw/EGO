let showSettingsModal = $state(false);
let showUserSettingsModal = $state(false);
let showUserStatsModal = $state(false);
let isThinking = $state(false);
export const uiStore = {
	get showSettingsModal() {
		return showSettingsModal;
	},
	get showUserSettingsModal() {
		return showUserSettingsModal;
	},
	get showUserStatsModal() {
		return showUserStatsModal;
	},
	get isThinking() {
		return isThinking;
	}
};
export function setShowSettingsModal(value: boolean) {
	showSettingsModal = value;
}
export function setShowUserSettingsModal(value: boolean) {
	showUserSettingsModal = value;
}
export function setShowUserStatsModal(value: boolean) {
	showUserStatsModal = value;
}
export function setIsThinking(value: boolean) {
	isThinking = value;
}

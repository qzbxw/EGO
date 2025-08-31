let memoryEnabled = $state<boolean>(true);
export const memoryStore = {
    get memoryEnabled() { return memoryEnabled; },
    setMemoryEnabled(enabled: boolean) {
        memoryEnabled = enabled;
    }
};
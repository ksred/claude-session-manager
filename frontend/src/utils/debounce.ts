export function createDebouncedInvalidator(delay: number = 5000) {
  const pendingInvalidations = new Map<string, NodeJS.Timeout>();
  
  return {
    debounceInvalidation: (key: string, invalidateFn: () => void) => {
      // Clear any existing timeout for this key
      const existingTimeout = pendingInvalidations.get(key);
      if (existingTimeout) {
        clearTimeout(existingTimeout);
      }
      
      // Set a new timeout
      const timeout = setTimeout(() => {
        invalidateFn();
        pendingInvalidations.delete(key);
      }, delay);
      
      pendingInvalidations.set(key, timeout);
    },
    
    clearAll: () => {
      pendingInvalidations.forEach(timeout => clearTimeout(timeout));
      pendingInvalidations.clear();
    }
  };
}
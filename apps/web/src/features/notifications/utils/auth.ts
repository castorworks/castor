export function shouldEnableNotificationQueries(isAuthenticated: boolean, isLoading: boolean) {
  return isAuthenticated && !isLoading;
}

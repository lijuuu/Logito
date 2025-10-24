/**
 * Converts a local datetime-local input value to UTC ISO string
 * @param localDateTimeString - The datetime-local input value (YYYY-MM-DDTHH:mm)
 * @returns UTC ISO string or empty string if input is empty
 */
export function convertLocalToUTC(localDateTimeString: string): string {
  if (!localDateTimeString) {
    return '';
  }

  // Create a Date object from the local datetime string
  // The datetime-local input provides a string in format "YYYY-MM-DDTHH:mm"
  // We need to treat this as local time and convert to UTC
  const localDate = new Date(localDateTimeString);

  // Return ISO string which is in UTC
  return localDate.toISOString();
}

/**
 * Converts a UTC ISO string to local datetime-local input value
 * @param utcIsoString - The UTC ISO string
 * @returns Local datetime-local input value (YYYY-MM-DDTHH:mm) or empty string if input is empty
 */
export function convertUTCToLocal(utcIsoString: string): string {
  if (!utcIsoString) {
    return '';
  }

  // Create a Date object from the UTC ISO string
  const utcDate = new Date(utcIsoString);

  // Get local time components
  const year = utcDate.getFullYear();
  const month = String(utcDate.getMonth() + 1).padStart(2, '0');
  const day = String(utcDate.getDate()).padStart(2, '0');
  const hours = String(utcDate.getHours()).padStart(2, '0');
  const minutes = String(utcDate.getMinutes()).padStart(2, '0');

  // Return in datetime-local format
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

/**
 * Formats a UTC ISO string for display in local time
 * @param utcIsoString - The UTC ISO string
 * @returns Formatted local time string
 */
export function formatUTCToLocalDisplay(utcIsoString: string): string {
  if (!utcIsoString) {
    return '';
  }

  const utcDate = new Date(utcIsoString);
  return utcDate.toLocaleString();
}

/**
 * Gets current local time in datetime-local format
 * @returns Current local time in YYYY-MM-DDTHH:mm format
 */
export function getCurrentLocalTime(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  const hours = String(now.getHours()).padStart(2, '0');
  const minutes = String(now.getMinutes()).padStart(2, '0');

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

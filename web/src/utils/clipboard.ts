/**
 * Copies text to the clipboard across all browser environments,
 * supporting both secure contexts (HTTPS / localhost via navigator.clipboard)
 * and non-secure contexts (plain HTTP over IP addresses via document.execCommand).
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  if (!text) return false;

  // 1. Try modern navigator.clipboard if available (secure contexts: HTTPS or localhost)
  if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch (err) {
      console.warn('navigator.clipboard.writeText failed, attempting execCommand fallback:', err);
    }
  }

  // 2. Universal fallback using a temporary hidden textarea and document.execCommand('copy')
  try {
    const textArea = document.createElement('textarea');
    textArea.value = text;

    // Prevent scrolling and layout shifts
    textArea.style.position = 'fixed';
    textArea.style.top = '0';
    textArea.style.left = '0';
    textArea.style.width = '2em';
    textArea.style.height = '2em';
    textArea.style.padding = '0';
    textArea.style.border = 'none';
    textArea.style.outline = 'none';
    textArea.style.boxShadow = 'none';
    textArea.style.background = 'transparent';
    textArea.style.opacity = '0';
    textArea.style.zIndex = '-9999';
    textArea.setAttribute('readonly', '');

    document.body.appendChild(textArea);
    textArea.focus({ preventScroll: true });
    textArea.select();
    textArea.setSelectionRange(0, textArea.value.length);

    const successful = document.execCommand('copy');
    document.body.removeChild(textArea);

    return successful;
  } catch (err) {
    console.error('execCommand copy fallback failed:', err);
    return false;
  }
}

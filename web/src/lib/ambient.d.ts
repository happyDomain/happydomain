// Types for what the project uses without the packages describing it.

// html-escaper ships no types of its own.
declare module "html-escaper" {
    export function escape(s: string): string;
    export function unescape(s: string): string;
}

// The captcha widgets are loaded at runtime from their provider, and each
// exposes itself as a global. Only the two calls the widget makes are
// described here.

// hCaptcha and Turnstile hand back a string, reCAPTCHA a number.
type CaptchaWidgetId = string | number;

interface CaptchaProviderGlobal {
    render(
        container: HTMLElement,
        options: { sitekey: string; callback: (token: string) => void },
    ): CaptchaWidgetId;
    reset(widgetId?: CaptchaWidgetId): void;
}

declare const hcaptcha: CaptchaProviderGlobal;
declare const grecaptcha: CaptchaProviderGlobal;
declare const turnstile: CaptchaProviderGlobal;

// Altcha registers a custom element, whose reset() is the only part used.
interface AltchaWidgetElement extends HTMLElement {
    reset?(): void;
}

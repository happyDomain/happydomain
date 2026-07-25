// downloadBlob triggers a client-side file download. An already-built Blob is
// used as-is (it carries its own type), avoiding a full in-memory copy of
// payloads that can be large, such as a database backup.
export function downloadBlob(content: string | Blob, filename: string, mime: string) {
    const blob = content instanceof Blob ? content : new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

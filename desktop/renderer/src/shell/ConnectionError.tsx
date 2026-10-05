export function ConnectionError({ onRetry }: { onRetry: () => void }) {
    return (
        <div className="connection-error" role="alert">
            <p>Can’t reach backend</p>
            <button type="button" onClick={onRetry}>
                Retry
            </button>
        </div>
    );
}

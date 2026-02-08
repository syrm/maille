import { useSignal } from '@preact/signals';
import { useEffect } from 'preact/hooks';

export function useFetch<T>(url: string) {
    const data = useSignal<T | null>(null);
    const loading = useSignal(true);
    const error = useSignal<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        fetch(url, { signal: controller.signal })
            .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
            .then(d => { data.value = d; })
            .catch(e => { if (e.name !== 'AbortError') error.value = e.message; })
            .finally(() => { loading.value = false; });

        return () => controller.abort();
    }, [url]);

    return { data, loading, error };
}

import { useFetch } from './useFetch';
import { BudgetSummary } from '../domain/budget';

export function useDashboard() {
    const { data: budgetSummary, loading, error } = useFetch<BudgetSummary>('http://localhost:13000/stats');
    return { budgetSummary, loading, error };
}

import { useDashboard } from '../hooks/useDashboard';

export function RecentTransaction() {
    const { budgetSummary, loading, error } = useDashboard();

    if (loading.value) return <p>Chargement...</p>;
    if (error.value) return <p>Erreur: {error.value}</p>;
    if (!budgetSummary.value) return <p>Aucune donnée.</p>;

    return (
        <div class="card">
            <div class="card-header">
                <div>
                    <div class="card-title">Transactions récentes</div>
                    <div class="card-sub">Dernières opérations</div>
                </div>
            </div>
            <div id="txList">
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--green-bg)">💰</div>
                    <div><div class="tx-name">Salaire Février</div><div class="tx-cat">Revenus · Virement</div></div>
                    <div class="tx-right"><div class="tx-amt inc">+2 850,00 €</div><div class="tx-date">5 fév.</div></div>
                </div>
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--red-bg)">🏠</div>
                    <div><div class="tx-name">Loyer</div><div class="tx-cat">Logement · Prélèvement</div></div>
                    <div class="tx-right"><div class="tx-amt exp">−780,00 €</div><div class="tx-date">5 fév.</div></div>
                </div>
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--amber-bg)">🛒</div>
                    <div><div class="tx-name">Carrefour</div><div class="tx-cat">Alimentation · CB</div></div>
                    <div class="tx-right"><div class="tx-amt exp">−67,42 €</div><div class="tx-date">4 fév.</div></div>
                </div>
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--purple-bg)">🎵</div>
                    <div><div class="tx-name">Spotify Premium</div><div class="tx-cat">Loisirs · Prélèvement</div></div>
                    <div class="tx-right"><div class="tx-amt exp">−10,99 €</div><div class="tx-date">3 fév.</div></div>
                </div>
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--blue-bg)">🚄</div>
                    <div><div class="tx-name">SNCF</div><div class="tx-cat">Transport · CB</div></div>
                    <div class="tx-right"><div class="tx-amt exp">−45,00 €</div><div class="tx-date">2 fév.</div></div>
                </div>
                <div class="tx-item">
                    <div class="tx-icon" style="background:var(--cyan-bg)">📊</div>
                    <div><div class="tx-name">Virement PEA</div><div class="tx-cat">Investissement · Virement</div></div>
                    <div class="tx-right"><div class="tx-amt exp">−300,00 €</div><div class="tx-date">1 fév.</div></div>
                </div></div>
        </div>
    );
}

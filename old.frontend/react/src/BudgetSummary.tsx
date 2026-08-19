export function BudgetSummary() {
    return (
        <>
        <div className="balance-row anim d2">
            <div className="card bal-card main-bal">
                <div className="bal-label"><span className="dot" style={{ background: "var(--green)" }}></span> Solde global</div>
                <div className="bal-value">87 432,60 €</div>
                <span className="badge up">↑ +3,2 % ce mois</span>
            </div>
            <div className="card bal-card">
                <div className="bal-label"><span className="dot" style={{ background: "var(--blue)" }}></span> Comptes courants</div>
                <div className="bal-value">12 840,25 €</div>
                <span className="badge up">↑ +540 €</span>
            </div>
            <div className="card bal-card">
                <div className="bal-label"><span className="dot" style={{ background: "var(--purple)" }}></span> Épargne</div>
                <div className="bal-value">38 210,00 €</div>
                <span className="badge up">↑ +1,1 %</span>
            </div>
            <div className="card bal-card">
                <div className="bal-label"><span className="dot" style={{ background: "var(--amber)" }}></span> Investissements</div>
                <div className="bal-value">36 382,35 €</div>
                <span className="badge down">↓ −0,8 %</span>
            </div>
        </div>
        </>
    );
}

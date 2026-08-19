import type {Component} from "solid-js";

export const BudgetSummary: Component = () => {
    return (
        <div class="balance-row anim d2">
            <div class="card bal-card main-bal">
                <div class="bal-label"><span class="dot" style="background:var(--green)"></span> Solde global</div>
                <div class="bal-value">123</div>
                <span class="badge up">↑ +3,2 % ce mois</span>
            </div>
            <div class="card bal-card">
                <div class="bal-label"><span class="dot" style="background:var(--blue)"></span> Comptes courants</div>
                <div class="bal-value">123</div>
                <span class="badge up">↑ +540 €</span>
            </div>
            <div class="card bal-card">
                <div class="bal-label"><span class="dot" style="background:var(--purple)"></span> Épargne</div>
                <div class="bal-value">123</div>
                <span class="badge up">↑ +1,1 %</span>
            </div>
            <div class="card bal-card">
                <div class="bal-label"><span class="dot" style="background:var(--amber)"></span> Investissements</div>
                <div class="bal-value">123</div>
                <span class="badge down">↓ −0,8 %</span>
            </div>
        </div>
    );
}

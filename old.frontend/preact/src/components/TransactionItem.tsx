import {Transaction} from "../domain/transaction";

export function TransactionItem(transaction: Transaction) {
    return (
        <div class="tx-item">
            <div class="tx-icon" style="background:var(--green-bg)">💰</div>
            <div>
                <div class="tx-name">{transaction.Annotation}</div>
                <div class="tx-cat">{transaction.Account}</div>
            </div>
            <div class="tx-right">
                <div class="tx-amt inc">{transaction.Amount} {transaction.Currency}</div>
                <div class="tx-date">{transaction.Date}</div>
            </div>
        </div>
    )
}

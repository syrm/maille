import { BudgetSummary } from "../../components/BudgetSummary";
import { RecentTransaction } from "../../components/RecentTransaction";

export function Home() {
	return (
		<>
			<BudgetSummary />
			<div class="row-3 anim d4">
				<RecentTransaction />
			</div>

			<div><a href="/test">Test</a></div>
		</>
	);
}

import { render } from 'preact';
import { LocationProvider, Router, Route } from 'preact-iso';

import { Header } from './components/Header';
import { Home } from './pages/Home';
import { Test } from './pages/Test';
import { NotFound } from './pages/_404';
import { Menu } from "./components/Menu";
import './style.css';

export function App() {
	return (
		<LocationProvider>
			<Menu />
			<main class="main">
				<Header />
				<Router>
					<Route path="/" component={Home} />
					<Route path="/test" component={Test} />
					<Route default component={NotFound} />
				</Router>
			</main>
		</LocationProvider>
	);
}

render(<App />, document.body);

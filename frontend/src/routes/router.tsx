import { createRootRoute, createRoute, createRouter, Link } from "@tanstack/react-router";
import { AppShell } from "../components/AppShell";
import { HomePage } from "../pages/HomePage";
import { AboutPage } from "../pages/AboutPage";

const rootRoute = createRootRoute({
  component: AppShell,
  notFoundComponent: () => (
    <section>
      <h1>Page not found</h1>
      <Link to="/">Back to home</Link>
    </section>
  ),
  errorComponent: ({ reset }) => (
    <section role="alert">
      <h1>Something went wrong</h1>
      <button onClick={reset}>Try again</button>
    </section>
  ),
});
const homeRoute = createRoute({ getParentRoute: () => rootRoute, path: "/", component: HomePage });
const aboutRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/about",
  component: AboutPage,
});
export const router = createRouter({
  routeTree: rootRoute.addChildren([homeRoute, aboutRoute]),
  scrollRestoration: true,
});
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

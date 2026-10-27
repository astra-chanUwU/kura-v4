import { createRootRoute, createRoute, createRouter, Link } from "@tanstack/react-router";
import { AppShell } from "../components/AppShell";
import { AuthPage } from "../pages/AuthPage";
import { HomePage } from "../pages/HomePage";
import { ListPage } from "../pages/ListPage";
import { MePage } from "../pages/MePage";
import { AdminPage } from "../pages/AdminPage";
import { HiddenPostsPage } from "../pages/HiddenPostsPage";
import { HistoryPage } from "../pages/HistoryPage";
import { PostPage } from "../pages/PostPage";
import { ProfilePage } from "../pages/ProfilePage";
import { RecoverPage } from "../pages/RecoverPage";

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
const homeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: HomePage,
  // The board's applied filter lives in the URL (?tags=a,b&exclude=c&q=x)
  // so Back/Forward restore the search you were exploring.
  validateSearch: (
    search: Record<string, unknown>,
  ): { tags?: string; exclude?: string; q?: string } => ({
    tags: typeof search.tags === "string" && search.tags !== "" ? search.tags : undefined,
    exclude:
      typeof search.exclude === "string" && search.exclude !== "" ? search.exclude : undefined,
    q: typeof search.q === "string" && search.q !== "" ? search.q : undefined,
  }),
});
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: () => <AuthPage mode="login" />,
});
const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/register",
  component: () => <AuthPage mode="register" />,
});
const meRoute = createRoute({ getParentRoute: () => rootRoute, path: "/me", component: MePage });
const adminRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/admin",
  component: AdminPage,
});
const hiddenRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/admin/hidden",
  component: HiddenPostsPage,
});
const historyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/admin/history",
  component: HistoryPage,
});
const recoverRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/recover",
  component: RecoverPage,
});
const postRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/posts/$id",
  component: () => {
    const { id } = postRoute.useParams();
    return <PostPage id={id} />;
  },
});
const profileRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/u/$username",
  component: () => {
    const { username } = profileRoute.useParams();
    return <ProfilePage username={username} />;
  },
});
const listRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/lists/$id",
  component: () => {
    const { id } = listRoute.useParams();
    return <ListPage id={id} />;
  },
});
export const router = createRouter({
  routeTree: rootRoute.addChildren([
    homeRoute,
    loginRoute,
    registerRoute,
    recoverRoute,
    meRoute,
    adminRoute,
    hiddenRoute,
    historyRoute,
    postRoute,
    profileRoute,
    listRoute,
  ]),
  scrollRestoration: true,
});
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

import { createRootRoute, createRoute, createRouter, Link } from "@tanstack/react-router";
import { AppShell } from "../components/AppShell";
import { validateBoardSearch } from "../domain/boardFilter";
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
  validateSearch: validateBoardSearch,
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
  // Opened from the board, a post carries the board's filter so
  // Previous/Next step through that filtered sequence.
  validateSearch: validateBoardSearch,
  component: () => {
    const { id } = postRoute.useParams();
    const search = postRoute.useSearch();
    return <PostPage id={id} search={search} />;
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

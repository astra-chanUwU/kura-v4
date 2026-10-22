import { useSite } from "../hooks/useSite";
import styles from "./HomePage.module.css";

export function HomePage() {
  const { site, loading, error, retry } = useSite();
  return (
    <section aria-labelledby="page-title">
      <p className={styles.eyebrow}>THE NEXT CHAPTER</p>
      <h1 id="page-title" className={styles.title}>
        A home for
        <br />
        your collection<span>.</span>
      </h1>
      <p className={styles.intro}>
        Images worth keeping. Details worth remembering.
        <br />A little space to make them yours.
      </p>
      <div className={styles.panel}>
        <div className={styles.illustration} aria-hidden="true">
          <div className={styles.frameBack} />
          <div className={styles.frameFront}>
            <span>蔵</span>
          </div>
        </div>
        <div className={styles.panelText}>
          <p className={styles.label}>A FRESH START</p>
          <h2>Your library starts here.</h2>
          <p>
            This is the beginning of Kura V4. Browsing, uploads and collections are coming next.
          </p>
          {loading && (
            <p role="status" className={styles.status}>
              Connecting to your space…
            </p>
          )}
          {error && (
            <div role="alert" className={styles.error}>
              <p>We couldn’t connect to your space.</p>
              <button onClick={retry}>Try again</button>
            </div>
          )}
          {site && !error && (
            <p role="status" className={styles.status}>
              <span className={styles.dot} />
              {site.name} is connected
            </p>
          )}
        </div>
      </div>
    </section>
  );
}

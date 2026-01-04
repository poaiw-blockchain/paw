import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function DevnetBanner() {
  return (
    <div className={styles.devnetBanner}>
      <span className={styles.devnetBadge}>DEVNET</span>
      <span>This is an invite-only development network. Tokens have no value.</span>
      <a href="https://poaiw.org" target="_blank" rel="noopener" className={styles.devnetLink}>Learn More →</a>
    </div>
  );
}

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <div className={styles.networkBadge}>Devnet Live</div>
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro">
            Get Started
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            to="#testnet-resources">
            Testnet Resources
          </Link>
        </div>
      </div>
    </header>
  );
}

function TestnetResources() {
  return (
    <section id="testnet-resources" className={styles.resourcesSection}>
      <div className="container">
        <Heading as="h2" className={styles.sectionTitle}>Devnet Resources</Heading>
        <div className={styles.resourcesGrid}>
          <a href="https://artifacts.poaiw.org" target="_blank" rel="noopener" className={styles.resourceCard}>
            <h3>Artifacts & Binaries</h3>
            <p>Pre-built binaries, genesis files, and configuration files</p>
          </a>
          <Link to="/faucet" className={styles.resourceCard}>
            <h3>Faucet</h3>
            <p>Get test PAW tokens to interact with the network</p>
          </Link>
          <a href="https://grafana.poaiw.org" target="_blank" rel="noopener" className={styles.resourceCard}>
            <h3>Network Status</h3>
            <p>Real-time monitoring and metrics dashboard</p>
          </a>
          <a href="https://explorer.poaiw.org" target="_blank" rel="noopener" className={styles.resourceCard}>
            <h3>Block Explorer</h3>
            <p>Explore blocks, transactions, and accounts</p>
          </a>
        </div>
      </div>
    </section>
  );
}


export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title="PAW Blockchain - Devnet"
      description="Verifiable AI Compute with Integrated DEX and Oracle - Development Network">
      <DevnetBanner />
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <TestnetResources />
      </main>
    </Layout>
  );
}

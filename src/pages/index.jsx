import React from 'react';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';

export default function Home() {
  return (
    <Layout
      title="TARINIO Documentation"
      description="Enterprise-grade self-hosted WAF platform documentation">
      <main className="hero-page">
        <section className="hero-panel">
          <p className="eyebrow">TARINIO 2.0.2</p>
          <h1>Документация TARINIO</h1>
          <p className="hero-copy">
            Единый портал по развёртыванию, эксплуатации, обновлениям, HA, PostgreSQL,
            observability и benchmark-сценариям для TARINIO WAF.
          </p>
          <div className="hero-actions">
            <Link className="button button--primary button--lg" to="/ru/">
              Открыть русскую wiki
            </Link>
            <Link className="button button--secondary button--lg" to="/en/">
              Open English docs
            </Link>
            <Link className="button button--secondary button--lg" to="/docs-overview">
              Открыть навигатор
            </Link>
          </div>
        </section>
      </main>
    </Layout>
  );
}

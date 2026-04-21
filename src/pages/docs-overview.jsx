import React from 'react';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';

export default function DocsOverviewPage() {
  return (
    <Layout title="Навигатор документации" description="Маршруты по документации TARINIO">
      <main className="overview-page">
        <h1>Навигатор документации</h1>
        <p>
          Эта страница помогает быстро понять, куда идти дальше: на внедрение, эксплуатацию,
          восстановление после сбоев, hardening или оценку enterprise-сценариев.
        </p>

        <div className="card-grid">
          <Link className="portal-card" to="/ru/navigator/">
            <h2>Русский навигатор</h2>
            <p>Маршруты чтения по задачам: внедрение, эксплуатация, доверие, DR и сопровождение.</p>
          </Link>
          <Link className="portal-card" to="/en/navigator/">
            <h2>English navigator</h2>
            <p>Task-based reading paths for deployment, operations, recovery, and validation.</p>
          </Link>
          <Link className="portal-card" to="/ru/compatibility-matrix/">
            <h2>Совместимость и sizing</h2>
            <p>Начни отсюда перед первым production rollout или HA-развёртыванием.</p>
          </Link>
          <Link className="portal-card" to="/ru/troubleshooting/">
            <h2>Troubleshooting</h2>
            <p>Быстрый переход к типовым симптомам, причинам и рабочим действиям оператора.</p>
          </Link>
        </div>
      </main>
    </Layout>
  );
}
